package controller

import "testing"

func TestAppendWAFIPWhitelistAddsUniqueIPs(t *testing.T) {
	next, added, existing, err := appendWAFIPWhitelist("192.0.2.1\n2001:db8::/32", []string{
		"192.0.2.1",
		"192.0.2.2",
		"2001:db8::1",
		"2001:db9::1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 2 || added[0] != "192.0.2.2" || added[1] != "2001:db9::1" {
		t.Fatalf("added = %#v", added)
	}
	if len(existing) != 2 || existing[0] != "192.0.2.1" || existing[1] != "2001:db8::1" {
		t.Fatalf("existing = %#v", existing)
	}
	want := "192.0.2.1\n2001:db8::/32\n192.0.2.2\n2001:db9::1"
	if next != want {
		t.Fatalf("next = %q, want %q", next, want)
	}
}

func TestAppendWAFIPWhitelistRejectsInvalidIP(t *testing.T) {
	if _, _, _, err := appendWAFIPWhitelist("", []string{"not-an-ip"}); err == nil {
		t.Fatal("invalid IP must be rejected")
	}
}
