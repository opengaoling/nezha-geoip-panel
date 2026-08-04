package controller

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/nezhahq/nezha/model"
)

// M10 regression: a PATCH /setting payload that omits "enable_mcp" must
// preserve the current value. With EnableMCP as a plain bool + omitempty,
// any partial update silently set EnableMCP=false and tripped the MCP
// kill switch (PurgeTransferEntries + RevokeStreamsForPurpose +
// CancelAllMCPInflight). Switching to *bool makes "field absent" a real
// signal at decode time.
func TestSettingForm_OmittedEnableMCPLeavesConfigUnchanged(t *testing.T) {
	body := []byte(`{"site_name":"X"}`)
	var sf model.SettingForm
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&sf); err != nil {
		t.Fatal(err)
	}
	if sf.EnableMCP != nil {
		t.Fatalf("EnableMCP must be nil when JSON omits the key, got %v", sf.EnableMCP)
	}
}

func TestSettingForm_OmittedJWTTimeoutLeavesConfigUnchanged(t *testing.T) {
	body := []byte(`{"site_name":"X"}`)
	var sf model.SettingForm
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&sf); err != nil {
		t.Fatal(err)
	}
	if sf.JWTTimeout != nil {
		t.Fatalf("JWTTimeout must be nil when JSON omits the key, got %v", *sf.JWTTimeout)
	}
}

func TestSettingForm_OmittedHideNewServersForGuestLeavesConfigUnchanged(t *testing.T) {
	body := []byte(`{"site_name":"X"}`)
	var sf model.SettingForm
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&sf); err != nil {
		t.Fatal(err)
	}
	if sf.HideNewServersForGuest != nil {
		t.Fatalf("HideNewServersForGuest must be nil when JSON omits the key, got %v", sf.HideNewServersForGuest)
	}
}

func TestSettingForm_ExplicitHideNewServersForGuestDecodes(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value bool
	}{
		{name: "enabled", value: true},
		{name: "disabled", value: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte(`{"hide_new_servers_for_guest":` + jsonBool(tc.value) + `}`)
			var sf model.SettingForm
			if err := json.NewDecoder(bytes.NewReader(body)).Decode(&sf); err != nil {
				t.Fatal(err)
			}
			if sf.HideNewServersForGuest == nil || *sf.HideNewServersForGuest != tc.value {
				t.Fatalf("HideNewServersForGuest must be *%v, got %v", tc.value, sf.HideNewServersForGuest)
			}
		})
	}
}

func jsonBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func TestSettingForm_ExplicitJWTTimeoutDecodes(t *testing.T) {
	body := []byte(`{"jwt_timeout":168}`)
	var sf model.SettingForm
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&sf); err != nil {
		t.Fatal(err)
	}
	if sf.JWTTimeout == nil || *sf.JWTTimeout != 168 {
		t.Fatalf("JWTTimeout must be *168, got %v", sf.JWTTimeout)
	}
}

func TestSettingForm_ExplicitEnableMCPTrueDecodes(t *testing.T) {
	body := []byte(`{"enable_mcp":true}`)
	var sf model.SettingForm
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&sf); err != nil {
		t.Fatal(err)
	}
	if sf.EnableMCP == nil || !*sf.EnableMCP {
		t.Fatalf("EnableMCP must be *true, got %v", sf.EnableMCP)
	}
}

func TestSettingForm_ExplicitEnableMCPFalseDecodes(t *testing.T) {
	body := []byte(`{"enable_mcp":false}`)
	var sf model.SettingForm
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&sf); err != nil {
		t.Fatal(err)
	}
	if sf.EnableMCP == nil || *sf.EnableMCP {
		t.Fatalf("EnableMCP must be *false, got %v", sf.EnableMCP)
	}
}

// updateMCPEnableFromForm is the resolver helper: nil = keep current,
// non-nil = use the explicit value. Kept as a small pure function so the
// kill-switch wiring stays trivial to audit.
func TestUpdateMCPEnableFromForm_NilKeepsCurrent(t *testing.T) {
	_, w := newRecorderCtxForMCPSettingTest(t)
	prev := true
	got := resolveSettingEnableMCP(nil, prev)
	if got != prev {
		t.Fatalf("nil form value must keep current=%v, got %v", prev, got)
	}
	if w.Code != 200 {
		t.Fatal("resolver must not write to response")
	}
}

func TestUpdateMCPEnableFromForm_NonNilOverrides(t *testing.T) {
	f := false
	if got := resolveSettingEnableMCP(&f, true); got != false {
		t.Fatal("explicit *false must override current=true")
	}
	tr := true
	if got := resolveSettingEnableMCP(&tr, false); got != true {
		t.Fatal("explicit *true must override current=false")
	}
}

func newRecorderCtxForMCPSettingTest(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}
