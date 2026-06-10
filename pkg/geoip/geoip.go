package geoip

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	maxminddb "github.com/oschwald/maxminddb-golang"
)

//go:embed geoip.db
var db []byte

var (
	countryDBPaths = []string{
		"geoip/geoip.db",
		"/dashboard/geoip/geoip.db",
		"data/geoip.db",
		"/dashboard/data/geoip.db",
		"pkg/geoip/geoip.db",
	}
	asnDBPaths = []string{
		"geoip/asn.mmdb",
		"geoip/GeoLite2-ASN.mmdb",
		"/dashboard/geoip/asn.mmdb",
		"/dashboard/geoip/GeoLite2-ASN.mmdb",
		"data/asn.mmdb",
		"data/GeoLite2-ASN.mmdb",
		"/dashboard/data/asn.mmdb",
		"/dashboard/data/GeoLite2-ASN.mmdb",
	}

	dbOnce = sync.OnceValues(func() (*maxminddb.Reader, error) {
		return openDB("NZ_GEOIP_DB", countryDBPaths, db)
	})

	asnDBOnce = sync.OnceValues(func() (*maxminddb.Reader, error) {
		return openDB("NZ_GEOIP_ASN_DB", asnDBPaths, nil)
	})
)

const (
	defaultCountryDBURL = "https://raw.githubusercontent.com/P3TERX/GeoLite.mmdb/download/GeoLite2-Country.mmdb"
	defaultASNDBURL     = "https://raw.githubusercontent.com/P3TERX/GeoLite.mmdb/download/GeoLite2-ASN.mmdb"
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

func EnsureDatabases() error {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("NZ_GEOIP_DOWNLOAD")), "0") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("NZ_GEOIP_DOWNLOAD")), "false") {
		return nil
	}

	var errs []error
	if err := ensureDBFile("GeoIP Country", "NZ_GEOIP_DB", "NZ_GEOIP_DB_URL", defaultCountryDBURL, countryDBPaths[0], countryDBPaths); err != nil {
		errs = append(errs, err)
	}
	if err := ensureDBFile("GeoIP ASN", "NZ_GEOIP_ASN_DB", "NZ_GEOIP_ASN_DB_URL", defaultASNDBURL, asnDBPaths[0], asnDBPaths); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
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

func ensureDBFile(name, pathEnv, urlEnv, defaultURL, defaultPath string, fallbackPaths []string) error {
	path := strings.TrimSpace(os.Getenv(pathEnv))
	envPath := path != ""
	if path == "" {
		path = defaultPath
	}
	hasUsableDB := dbFileUsable(path) || (!envPath && anyDBFileUsable(fallbackPaths))

	url := firstNonEmpty(os.Getenv(urlEnv), defaultURL)
	if url == "" {
		if hasUsableDB {
			return nil
		}
		return fmt.Errorf("%s database download URL is empty", name)
	}
	if err := downloadDBFile(path, url); err != nil {
		if hasUsableDB {
			return nil
		}
		return fmt.Errorf("%s database: %w", name, err)
	}
	return nil
}

func anyDBFileUsable(paths []string) bool {
	for _, path := range paths {
		if dbFileUsable(path) {
			return true
		}
	}
	return false
}

func dbFileUsable(path string) bool {
	reader, err := openDBFile(path)
	if err != nil {
		return false
	}
	_ = reader.Close()
	return true
}

func downloadDBFile(path, url string) error {
	cleanPath := filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return err
	}

	tmpPath := cleanPath + ".tmp"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected HTTP status %s", resp.Status)
	}

	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if !dbFileUsable(tmpPath) {
		_ = os.Remove(tmpPath)
		return errors.New("downloaded database is empty")
	}
	return os.Rename(tmpPath, cleanPath)
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
