package model

import "testing"

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
