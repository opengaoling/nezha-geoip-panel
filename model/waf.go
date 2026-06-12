package model

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/nezhahq/nezha/pkg/utils"
	"gorm.io/gorm"
)

const (
	_ uint8 = iota
	WAFBlockReasonTypeLoginFail
	WAFBlockReasonTypeBruteForceToken
	WAFBlockReasonTypeAgentAuthFail
	WAFBlockReasonTypeManual
	WAFBlockReasonTypeBruteForceOauth2
)

const (
	BlockIDgRPC = -127 + iota
	BlockIDToken
	BlockIDUnknownUser
	BlockIDManual
)

type WAFApiMock struct {
	IP              string `json:"ip,omitempty"`
	BlockIdentifier int64  `json:"block_identifier,omitempty"`
	BlockReason     uint8  `json:"block_reason,omitempty"`
	BlockTimestamp  uint64 `json:"block_timestamp,omitempty"`
	Count           uint64 `json:"count,omitempty"`
}

type WAF struct {
	IP              []byte `gorm:"type:binary(16);primaryKey" json:"ip,omitempty"`
	BlockIdentifier int64  `gorm:"primaryKey" json:"block_identifier,omitempty"`
	BlockReason     uint8  `json:"block_reason,omitempty"`
	BlockTimestamp  uint64 `gorm:"index" json:"block_timestamp,omitempty"`
	Count           uint64 `json:"count,omitempty"`
}

var WAFIPWhitelistChecker func(ip string) bool

const (
	wafCleanCacheTTL     = 5 * time.Second
	wafCleanCacheMaxSize = 4096
)

var (
	wafCheckCleanCache   = newWAFTTLCache()
	wafUnblockCleanCache = newWAFTTLCache()
)

type wafTTLCache struct {
	mu    sync.Mutex
	items map[string]time.Time
}

func newWAFTTLCache() *wafTTLCache {
	return &wafTTLCache{items: make(map[string]time.Time)}
}

func (c *wafTTLCache) get(key string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt, ok := c.items[key]
	if !ok {
		return false
	}
	if now.Before(expiresAt) {
		return true
	}
	delete(c.items, key)
	return false
}

func (c *wafTTLCache) set(key string, now time.Time) {
	c.mu.Lock()
	if len(c.items) >= wafCleanCacheMaxSize {
		c.pruneLocked(now)
	}
	c.items[key] = now.Add(wafCleanCacheTTL)
	c.mu.Unlock()
}

func (c *wafTTLCache) delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

func (c *wafTTLCache) pruneLocked(now time.Time) {
	for key, expiresAt := range c.items {
		if !now.Before(expiresAt) || len(c.items) >= wafCleanCacheMaxSize {
			delete(c.items, key)
		}
	}
}

func (w *WAF) TableName() string {
	return "nz_waf"
}

func IsWAFIPWhitelisted(ip string) bool {
	return WAFIPWhitelistChecker != nil && WAFIPWhitelistChecker(ip)
}

func CheckIP(db *gorm.DB, ip string) error {
	if ip == "" {
		return nil
	}
	if IsWAFIPWhitelisted(ip) {
		return nil
	}
	ipBinary, err := utils.IPStringToBinary(ip)
	if err != nil {
		return err
	}
	cacheKey := wafCheckCacheKey(db, ipBinary)
	checkedAt := time.Now()
	if wafCheckCleanCache.get(cacheKey, checkedAt) {
		return nil
	}

	var blockTimestamp uint64
	result := db.Model(&WAF{}).Order("block_timestamp desc").Select("block_timestamp").Where("ip = ?", ipBinary).Limit(1).Find(&blockTimestamp)
	if result.Error != nil {
		return result.Error
	}

	// 检查是否未找到记录
	if result.RowsAffected < 1 {
		wafCheckCleanCache.set(cacheKey, checkedAt)
		return nil
	}

	var count uint64
	if err := db.Model(&WAF{}).Select("SUM(count)").Where("ip = ?", ipBinary).Scan(&count).Error; err != nil {
		return err
	}

	unixNow := time.Now().Unix()
	if powAdd(count, 4, blockTimestamp) > uint64(unixNow) {
		return errors.New("you were blocked by nezha WAF")
	}
	return nil
}

func UnblockIP(db *gorm.DB, ip string, uid int64) error {
	if ip == "" {
		return nil
	}
	if IsWAFIPWhitelisted(ip) {
		return nil
	}
	ipBinary, err := utils.IPStringToBinary(ip)
	if err != nil {
		return err
	}
	cacheKey := wafUnblockCacheKey(db, ipBinary, uid)
	now := time.Now()
	if wafUnblockCleanCache.get(cacheKey, now) {
		return nil
	}

	result := db.Unscoped().Delete(&WAF{}, "ip = ? and block_identifier = ?", ipBinary, uid)
	if result.Error != nil {
		return result.Error
	}
	wafUnblockCleanCache.set(cacheKey, now)
	return nil
}

func BatchUnblockIP(db *gorm.DB, ip []string) error {
	if len(ip) < 1 {
		return nil
	}
	ips := make([][]byte, 0, len(ip))
	for _, s := range ip {
		ipBinary, err := utils.IPStringToBinary(s)
		if err != nil {
			continue
		}
		ips = append(ips, ipBinary)
	}
	return db.Unscoped().Delete(&WAF{}, "ip in (?)", ips).Error
}

func BlockIP(db *gorm.DB, ip string, reason uint8, uid int64) error {
	if ip == "" {
		return nil
	}
	if IsWAFIPWhitelisted(ip) {
		return nil
	}
	ipBinary, err := utils.IPStringToBinary(ip)
	if err != nil {
		return err
	}
	wafCheckCleanCache.delete(wafCheckCacheKey(db, ipBinary))
	wafUnblockCleanCache.delete(wafUnblockCacheKey(db, ipBinary, uid))

	w := WAF{
		IP:              ipBinary,
		BlockIdentifier: uid,
	}
	now := uint64(time.Now().Unix())

	var count any
	if reason == WAFBlockReasonTypeManual {
		count = 99999
	} else {
		count = gorm.Expr("count + 1")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(&w).Attrs(WAF{
			BlockReason:    reason,
			BlockTimestamp: now,
		}).FirstOrCreate(&w).Error; err != nil {
			return err
		}
		return tx.Exec("UPDATE nz_waf SET count = ?, block_reason = ?, block_timestamp = ? WHERE ip = ? and block_identifier = ?", count, reason, now, ipBinary, uid).Error
	})
}

func wafCheckCacheKey(db *gorm.DB, ipBinary []byte) string {
	return fmt.Sprintf("%p", db) + "\x00" + string(ipBinary)
}

func wafUnblockCacheKey(db *gorm.DB, ipBinary []byte, uid int64) string {
	return wafCheckCacheKey(db, ipBinary) + "\x00" + strconv.FormatInt(uid, 10)
}

func powAdd(x, y, z uint64) uint64 {
	base := big.NewInt(0).SetUint64(x)
	exp := big.NewInt(0).SetUint64(y)
	result := big.NewInt(1)
	result.Exp(base, exp, nil)
	result.Add(result, big.NewInt(0).SetUint64(z))
	if !result.IsUint64() {
		return ^uint64(0) // return max uint64 value on overflow
	}
	ret := result.Uint64()
	return utils.IfOr(ret < z+3, z+3, ret)
}
