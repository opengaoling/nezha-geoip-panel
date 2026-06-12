package model

import (
	"errors"
	"testing"
	"time"

	"github.com/nezhahq/nezha/pkg/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestWAFIPWhitelistMatchesLineSeparatedIPAndCIDR(t *testing.T) {
	cfg := &Config{ConfigDashboard: ConfigDashboard{
		WAFIPWhitelist: "2001:db8::1\n192.0.2.0/24\n\n",
	}}

	if !cfg.IsWAFIPWhitelisted("2001:db8::1") {
		t.Fatal("exact IPv6 whitelist entry must match")
	}
	if !cfg.IsWAFIPWhitelisted("192.0.2.42") {
		t.Fatal("CIDR whitelist entry must match addresses inside the prefix")
	}
	if cfg.IsWAFIPWhitelisted("192.0.3.42") {
		t.Fatal("address outside the CIDR prefix must not match")
	}
}

func TestWAFIPWhitelistDoesNotTreatCommaAsSeparator(t *testing.T) {
	cfg := &Config{ConfigDashboard: ConfigDashboard{
		WAFIPWhitelist: "198.51.100.1,198.51.100.2",
	}}

	if cfg.IsWAFIPWhitelisted("198.51.100.1") || cfg.IsWAFIPWhitelisted("198.51.100.2") {
		t.Fatal("WAF whitelist is line-separated; comma-separated entries must not match")
	}
}

func TestCheckIPCleanCacheIsClearedByBlockIP(t *testing.T) {
	db := newWAFTestDB(t)
	ip := "203.0.113.8"

	if err := CheckIP(db, ip); err != nil {
		t.Fatalf("initial CheckIP: %v", err)
	}
	if err := BlockIP(db, ip, WAFBlockReasonTypeManual, BlockIDManual); err != nil {
		t.Fatalf("BlockIP: %v", err)
	}
	if err := CheckIP(db, ip); err == nil {
		t.Fatal("CheckIP must see a block inserted after a clean-cache hit")
	}
}

func TestUnblockIPCleanCacheAvoidsRepeatedEmptyDelete(t *testing.T) {
	db := newWAFTestDB(t)
	ip := "203.0.113.9"

	if err := UnblockIP(db, ip, BlockIDgRPC); err != nil {
		t.Fatalf("initial UnblockIP: %v", err)
	}

	ipBinary, err := utils.IPStringToBinary(ip)
	if err != nil {
		t.Fatalf("IPStringToBinary: %v", err)
	}
	key := wafUnblockCacheKey(db, ipBinary, BlockIDgRPC)
	if !wafUnblockCleanCache.get(key, time.Now()) {
		t.Fatal("empty UnblockIP should cache the clean ip/block identifier pair")
	}
}

func TestUnblockIPSkipsWhitelistedIP(t *testing.T) {
	prev := WAFIPWhitelistChecker
	WAFIPWhitelistChecker = func(ip string) bool {
		return ip == "203.0.113.10"
	}
	t.Cleanup(func() {
		WAFIPWhitelistChecker = prev
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DryRun: true,
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.AddError(errors.New("database should not be touched"))

	if err := UnblockIP(db, "203.0.113.10", BlockIDgRPC); err != nil {
		t.Fatalf("whitelisted UnblockIP should return before database access: %v", err)
	}
}

func newWAFTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&WAF{}); err != nil {
		t.Fatalf("migrate WAF: %v", err)
	}
	return db
}
