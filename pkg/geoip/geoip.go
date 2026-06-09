package geoip

import (
	_ "embed"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	maxminddb "github.com/oschwald/maxminddb-golang"
)

//go:embed geoip.db
var db []byte

var (
	dbOnce = sync.OnceValues(func() (*maxminddb.Reader, error) {
		return openDB("NZ_GEOIP_DB", []string{
			"data/geoip.db",
			"/dashboard/data/geoip.db",
			"pkg/geoip/geoip.db",
		}, db)
	})

	asnDBOnce = sync.OnceValues(func() (*maxminddb.Reader, error) {
		return openDB("NZ_GEOIP_ASN_DB", []string{
			"data/asn.mmdb",
			"data/GeoLite2-ASN.mmdb",
			"/dashboard/data/asn.mmdb",
			"/dashboard/data/GeoLite2-ASN.mmdb",
		}, nil)
	})
)

type IPInfo struct {
	Country                      string `maxminddb:"country"`
	CountryName                  string `maxminddb:"country_name"`
	Continent                    string `maxminddb:"continent"`
	ContinentName                string `maxminddb:"continent_name"`
	ASName                       string `maxminddb:"as_name"`
	ASDomain                     string `maxminddb:"as_domain"`
	AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
	ISP                          string `maxminddb:"isp"`
	Organization                 string `maxminddb:"organization"`
	Org                          string `maxminddb:"org"`
}

type standardIPInfo struct {
	Country                      namedCodeRecord `maxminddb:"country"`
	RegisteredCountry            namedCodeRecord `maxminddb:"registered_country"`
	Continent                    namedCodeRecord `maxminddb:"continent"`
	AutonomousSystemOrganization string          `maxminddb:"autonomous_system_organization"`
}

type namedCodeRecord struct {
	ISOCode string            `maxminddb:"iso_code"`
	Code    string            `maxminddb:"code"`
	Names   map[string]string `maxminddb:"names"`
}

type LookupResult struct {
	CountryCode  string
	Continent    string
	Organization string
}

func Lookup(ip net.IP) (string, error) {
	info, err := LookupInfo(ip)
	if err != nil {
		return "", err
	}

	if info.CountryCode != "" {
		return strings.ToLower(info.CountryCode), nil
	} else if info.Continent != "" {
		return strings.ToLower(info.Continent), nil
	}
	return "", errors.New("IP not found")
}

func LookupInfo(ip net.IP) (*LookupResult, error) {
	if ip == nil {
		return nil, errors.New("invalid IP")
	}

	result := &LookupResult{}
	var firstErr error

	if db, err := dbOnce(); err == nil {
		if err := lookupInto(db, ip, result); err != nil {
			firstErr = err
		}
	} else {
		firstErr = err
	}

	if db, err := asnDBOnce(); err == nil {
		if err := lookupInto(db, ip, result); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if result.CountryCode != "" || result.Continent != "" || result.Organization != "" {
		return result, nil
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return nil, errors.New("IP not found")
}

func lookupInto(db *maxminddb.Reader, ip net.IP, result *LookupResult) error {
	var standard standardIPInfo
	standardErr := db.Lookup(ip, &standard)
	if standardErr == nil {
		applyStandardInfo(result, &standard)
	}

	var record IPInfo
	err := db.Lookup(ip, &record)
	if err != nil && standardErr != nil {
		return err
	}
	if err == nil {
		applyFlatInfo(result, &record)
	}
	return nil
}

func applyStandardInfo(result *LookupResult, record *standardIPInfo) {
	if result.CountryCode == "" {
		result.CountryCode = firstNonEmpty(record.Country.ISOCode, record.RegisteredCountry.ISOCode)
	}
	if result.Continent == "" {
		result.Continent = firstNonEmpty(record.Continent.Code, localizedName(record.Continent.Names))
	}
	if result.Organization == "" {
		result.Organization = record.AutonomousSystemOrganization
	}
}

func applyFlatInfo(result *LookupResult, record *IPInfo) {
	if result.CountryCode == "" {
		result.CountryCode = record.Country
	}
	if result.Continent == "" {
		result.Continent = record.Continent
	}
	if result.Organization == "" {
		result.Organization = firstNonEmpty(record.ASName, record.AutonomousSystemOrganization, record.Organization, record.Org, record.ISP, record.ASDomain)
	}
}

func openDB(envKey string, paths []string, fallback []byte) (*maxminddb.Reader, error) {
	if envPath := strings.TrimSpace(os.Getenv(envKey)); envPath != "" {
		return openDBFile(envPath)
	}

	for _, path := range paths {
		reader, err := openDBFile(path)
		if err == nil {
			return reader, nil
		}
	}

	if len(fallback) == 0 {
		return nil, os.ErrNotExist
	}
	return maxminddb.FromBytes(fallback)
}

func openDBFile(path string) (*maxminddb.Reader, error) {
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(b)) == "stub" {
		return nil, errors.New("geoip database is a stub")
	}
	return maxminddb.FromBytes(b)
}

func localizedName(names map[string]string) string {
	return firstNonEmpty(names["en"], names["zh-CN"], names["zh"], names["ja"])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
