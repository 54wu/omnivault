package api

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

// bridgeResult is the JSON envelope returned by NativeBridge.Request.
type bridgeResult struct {
	Status      int    `json:"status"`
	ContentType string `json:"content_type"`
	Filename    string `json:"filename"`
	DataB64     string `json:"data_b64"`
}

func decodeBridge(t *testing.T, raw string) bridgeResult {
	t.Helper()
	var r bridgeResult
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatalf("bad bridge envelope %q: %v", raw, err)
	}
	return r
}

func bridgeBody(t *testing.T, r bridgeResult) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(r.DataB64)
	if err != nil {
		t.Fatalf("bad base64 body: %v", err)
	}
	return data
}

// TestNativeBridge_Fields exercises the in-process bridge used by the UI:
// list fields, set a value, get it back, and confirm auth is applied.
func TestNativeBridge_Fields(t *testing.T) {
	env := setup(t)
	b := NewNativeBridge(env.server.Handler(), env.token)

	// Set a field through the bridge.
	setRaw := b.Request("PUT", "/vault/fields/identity.full_name",
		`{"Content-Type":"application/json"}`, b64of(`{"value":"Bridge Alice"}`))
	set := decodeBridge(t, setRaw)
	if set.Status != 200 {
		t.Fatalf("set: want 200, got %d: %s", set.Status, bridgeBody(t, set))
	}

	// List fields; the bridge authenticates as the in-process token.
	listRaw := b.Request("GET", "/vault/fields",
		`{"Authorization":"Bearer totally-bogus,but-overridden"}`, "")
	list := decodeBridge(t, listRaw)
	if list.Status != 200 {
		t.Fatalf("list: want 200, got %d: %s", list.Status, bridgeBody(t, list))
	}
	var fields []map[string]any
	if err := json.Unmarshal(bridgeBody(t, list), &fields); err != nil {
		t.Fatalf("list payload not json: %v", err)
	}
	found := false
	for _, f := range fields {
		if f["id"] == "identity.full_name" {
			found = true
		}
	}
	if !found {
		t.Fatalf("field not returned by bridge list: %+v", fields)
	}
}

// TestNativeBridge_Attachment round-trips an attachment upload and download
// through the bridge, including binary content.
func TestNativeBridge_Attachment(t *testing.T) {
	env := setup(t)
	b := NewNativeBridge(env.server.Handler(), env.token)

	if get := decodeBridge(t, b.Request("PUT", "/vault/fields/p1_identity.name",
		`{"Content-Type":"application/json"}`, b64of(`{"value":"Doc"}`))); get.Status != 200 {
		t.Fatalf("set field failed: %s", bridgeBody(t, get))
	}

	content := []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0x7f}
	upload := decodeBridge(t, b.Request("POST",
		"/vault/attachments?field=p1_identity.name",
		`{"X-Filename":"a.bin","Content-Type":"application/octet-stream"}`,
		base64.StdEncoding.EncodeToString(content)))
	if upload.Status != 201 {
		t.Fatalf("upload: want 201, got %d: %s", upload.Status, bridgeBody(t, upload))
	}
	var info map[string]any
	if err := json.Unmarshal(bridgeBody(t, upload), &info); err != nil {
		t.Fatalf("upload payload: %v", err)
	}
	id, _ := info["id"].(string)
	if id == "" {
		t.Fatalf("upload returned no id: %+v", info)
	}

	// Download and verify bytes round-trip.
	down := decodeBridge(t, b.Request("GET", "/vault/attachments/"+id,
		`{"Authorization":"Bearer x"}`, ""))
	if down.Status != 200 {
		t.Fatalf("download: want 200, got %d: %s", down.Status, bridgeBody(t, down))
	}
	got := bridgeBody(t, down)
	if string(got) != string(content) {
		t.Fatalf("download bytes mismatch: got %v want %v", got, content)
	}
}

// TestNativeBridge_ChangePassword verifies the bridge adopts the new session
// token so subsequent requests still authenticate.
func TestNativeBridge_ChangePassword(t *testing.T) {
	env := setup(t)
	b := NewNativeBridge(env.server.Handler(), env.token)

	cp := decodeBridge(t, b.Request("POST", "/vault/change-password",
		`{"Content-Type":"application/json"}`,
		b64of(`{"old_password":"`+testPassword+`","new_password":"new-password-4567"}`)))
	if cp.Status != 200 {
		t.Fatalf("change password: want 200, got %d: %s", cp.Status, bridgeBody(t, cp))
	}
	if b.token == env.token || b.token == "" {
		t.Fatalf("bridge did not adopt new token: %q", b.token)
	}

	// A protected write after the password change must still authenticate.
	if get := decodeBridge(t, b.Request("GET", "/vault/fields",
		`{"Authorization":"Bearer whatever"}`, "")); get.Status != 200 {
		t.Fatalf("post-change list failed: %s", bridgeBody(t, get))
	}
}

func b64of(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}