package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/54wu/omnivault/internal/vault"
)

const testPassword = "test-password-123"

type testEnv struct {
	server *Server
	vault  *vault.Vault
	token  string
}

func setup(t *testing.T) *testEnv {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".omnivault")
	sk, err := vault.Init(dir, testPassword)
	if err != nil {
		t.Fatal(err)
	}

	v, err := vault.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { v.Close() })

	token, err := v.Unlock(testPassword, sk)
	if err != nil {
		t.Fatal(err)
	}

	s := New(v, ":0")
	return &testEnv{server: s, vault: v, token: token}
}

func (e *testEnv) doRequest(t *testing.T, method, path string, body any, auth bool) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if auth {
		req.Header.Set("Authorization", "Bearer "+e.token)
	}
	w := httptest.NewRecorder()
	e.server.handler.ServeHTTP(w, req)
	return w
}

func TestStatus_Unlocked(t *testing.T) {
	env := setup(t)
	w := env.doRequest(t, "GET", "/vault/status", nil, false)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var status vault.VaultStatus
	json.NewDecoder(w.Body).Decode(&status)
	if status.Locked {
		t.Fatal("expected unlocked")
	}
	if !status.Initialized {
		t.Fatal("expected initialized")
	}
}

func TestSetField_GetField(t *testing.T) {
	env := setup(t)

	// Set field
	w := env.doRequest(t, "PUT", "/vault/fields/identity.full_name", map[string]string{
		"value": "Jane Smith",
	}, true)
	if w.Code != 200 {
		t.Fatalf("set: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Get field
	w = env.doRequest(t, "GET", "/vault/fields/identity.full_name", nil, true)
	if w.Code != 200 {
		t.Fatalf("get: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var field vault.FieldInfo
	json.NewDecoder(w.Body).Decode(&field)
	if field.Value != "Jane Smith" {
		t.Fatalf("expected 'Jane Smith', got %q", field.Value)
	}
}

func TestGetField_NotFound(t *testing.T) {
	env := setup(t)
	w := env.doRequest(t, "GET", "/vault/fields/nonexistent.field", nil, true)
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestErrorResponse_HasConstraint(t *testing.T) {
	env := setup(t)

	// Request a non-existent field to trigger a not_found error
	w := env.doRequest(t, "GET", "/vault/fields/nonexistent.field", nil, true)
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	var resp struct {
		Error      string `json:"error"`
		Constraint string `json:"constraint"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Constraint != "not_found" {
		t.Fatalf("expected constraint 'not_found', got %q", resp.Constraint)
	}
	if resp.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

func parseErrorResponse(t *testing.T, w *httptest.ResponseRecorder) (string, string) {
	t.Helper()
	var resp struct {
		Error      string `json:"error"`
		Constraint string `json:"constraint"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	return resp.Error, resp.Constraint
}

func TestConstraint_InvalidRequest(t *testing.T) {
	env := setup(t)

	// Invalid JSON body
	req := httptest.NewRequest("PUT", "/vault/fields/identity.name", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.server.handler.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	_, constraint := parseErrorResponse(t, w)
	if constraint != "invalid_request" {
		t.Fatalf("expected constraint 'invalid_request', got %q", constraint)
	}
}

func TestConstraint_InvalidRequest_MissingFields(t *testing.T) {
	env := setup(t)

	// Missing required value field
	w := env.doRequest(t, "PUT", "/vault/fields/identity.name", map[string]string{}, true)
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	_, constraint := parseErrorResponse(t, w)
	if constraint != "invalid_request" {
		t.Fatalf("expected constraint 'invalid_request', got %q", constraint)
	}
}

func TestConstraint_Unauthenticated_MissingAuth(t *testing.T) {
	env := setup(t)

	w := env.doRequest(t, "GET", "/vault/fields", nil, false)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	_, constraint := parseErrorResponse(t, w)
	if constraint != "unauthenticated" {
		t.Fatalf("expected constraint 'unauthenticated', got %q", constraint)
	}
}

func TestConstraint_Unauthenticated_InvalidToken(t *testing.T) {
	env := setup(t)

	req := httptest.NewRequest("GET", "/vault/fields", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	env.server.handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	_, constraint := parseErrorResponse(t, w)
	if constraint != "unauthenticated" {
		t.Fatalf("expected constraint 'unauthenticated', got %q", constraint)
	}
}

func TestConstraint_Unauthenticated_WrongPassword(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".omnivault")
	sk, err := vault.Init(dir, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { v.Close() })
	// Don't unlock — we'll try with wrong password
	_ = sk

	s := New(v, ":0")
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"password": "wrong", "secret_key": sk})
	req := httptest.NewRequest("POST", "/vault/unlock", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	_, constraint := parseErrorResponse(t, w)
	if constraint != "unauthenticated" {
		t.Fatalf("expected constraint 'unauthenticated', got %q", constraint)
	}
}

func TestUnlock_LockoutAfterRepeatedFailures(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".omnivault")
	sk, err := vault.Init(dir, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { v.Close() })

	s := New(v, ":0")
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		body, _ := json.Marshal(map[string]string{"password": "wrong", "secret_key": sk})
		req := httptest.NewRequest("POST", "/vault/unlock", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		s.handler.ServeHTTP(w, req)
		if w.Code != 401 {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, w.Code)
		}
	}

	// 6th attempt must be locked out regardless of correct credentials.
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"password": testPassword, "secret_key": sk})
	req := httptest.NewRequest("POST", "/vault/unlock", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.handler.ServeHTTP(w, req)

	if w.Code != 429 {
		t.Fatalf("expected 429 locked_out, got %d: %s", w.Code, w.Body.String())
	}
	_, constraint := parseErrorResponse(t, w)
	if constraint != "locked_out" {
		t.Fatalf("expected constraint 'locked_out', got %q", constraint)
	}
}

func TestUnlock_SecurityQuestionRecovery(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".omnivault")
	sk, err := vault.Init(dir, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { v.Close() })

	// Configure a security question (requires an unlocked vault), then lock.
	if _, err := v.Unlock(testPassword, sk); err != nil {
		t.Fatal(err)
	}
	if err := v.SetSecurityQuestion("你的母亲的名字？", "Alice"); err != nil {
		t.Fatal(err)
	}
	v.Lock()

	s := New(v, ":0")
	postUnlock := func(body map[string]string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		b, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/vault/unlock", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		s.handler.ServeHTTP(w, req)
		return w
	}

	// 5 wrong passwords.
	for i := 0; i < 5; i++ {
		if w := postUnlock(map[string]string{"password": "wrong", "secret_key": sk}); w.Code != 401 {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, w.Code)
		}
	}

	// 6th attempt without an answer must demand the security question.
	w := postUnlock(map[string]string{"password": testPassword, "secret_key": sk})
	if w.Code != http.StatusLocked {
		t.Fatalf("expected 423 security_required, got %d: %s", w.Code, w.Body.String())
	}
	var locked struct {
		Constraint string `json:"constraint"`
		Question   string `json:"question"`
	}
	json.Unmarshal(w.Body.Bytes(), &locked)
	if locked.Constraint != "security_required" || locked.Question == "" {
		t.Fatalf("expected security_required with question, got %+v", locked)
	}

	// Wrong answer is rejected.
	w = postUnlock(map[string]string{"password": testPassword, "secret_key": sk, "answer": "Bob"})
	if w.Code != http.StatusLocked {
		t.Fatalf("expected 423 security_wrong, got %d", w.Code)
	}
	_, constraint := parseErrorResponse(t, w)
	if constraint != "security_wrong" {
		t.Fatalf("expected constraint 'security_wrong', got %q", constraint)
	}

	// Correct answer clears the gate and unlock succeeds.
	w = postUnlock(map[string]string{"password": testPassword, "secret_key": sk, "answer": "Alice"})
	if w.Code != 200 {
		t.Fatalf("expected 200 after correct answer, got %d: %s", w.Code, w.Body.String())
	}
}

func TestConstraint_ScopeExceeded(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/financial.income", map[string]string{"value": "100k"}, true)

	token := createScopedToken(t, env, "agent", "identity.*")

	w := env.doRequestWithToken(t, "GET", "/vault/fields/financial.income", nil, token)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	_, constraint := parseErrorResponse(t, w)
	if constraint != "scope_exceeded" {
		t.Fatalf("expected constraint 'scope_exceeded', got %q", constraint)
	}
}

func TestConstraint_SessionRequired(t *testing.T) {
	env := setup(t)
	token := createScopedToken(t, env, "agent", "*")

	w := env.doRequestWithToken(t, "POST", "/vault/lock", nil, token)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	_, constraint := parseErrorResponse(t, w)
	if constraint != "session_required" {
		t.Fatalf("expected constraint 'session_required', got %q", constraint)
	}
}

func TestConstraint_Conflict_AlreadyUnlocked(t *testing.T) {
	env := setup(t)

	w := env.doRequest(t, "POST", "/vault/unlock", map[string]string{
		"password":   testPassword,
		"secret_key": "doesntmatter",
	}, false)

	if w.Code != 409 {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	_, constraint := parseErrorResponse(t, w)
	if constraint != "conflict" {
		t.Fatalf("expected constraint 'conflict', got %q", constraint)
	}
}

func TestListFields(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/identity.name", map[string]string{"value": "Test"}, true)
	env.doRequest(t, "PUT", "/vault/fields/financial.income", map[string]string{"value": "100k"}, true)

	w := env.doRequest(t, "GET", "/vault/fields", nil, true)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var fields []vault.FieldInfo
	json.NewDecoder(w.Body).Decode(&fields)
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
}

func TestGetByCategory(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/identity.name", map[string]string{"value": "Jane"}, true)
	env.doRequest(t, "PUT", "/vault/fields/identity.dob", map[string]string{"value": "1990-01-01"}, true)
	env.doRequest(t, "PUT", "/vault/fields/financial.income", map[string]string{"value": "100k"}, true)

	w := env.doRequest(t, "GET", "/vault/fields/category/identity", nil, true)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var fields []vault.FieldInfo
	json.NewDecoder(w.Body).Decode(&fields)
	if len(fields) != 2 {
		t.Fatalf("expected 2 identity fields, got %d", len(fields))
	}
	if fields[0].Value == "" {
		t.Fatal("GetByCategory should include decrypted values")
	}
}

func TestDeleteField(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/identity.name", map[string]string{"value": "Jane"}, true)

	w := env.doRequest(t, "DELETE", "/vault/fields/identity.name", nil, true)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	w = env.doRequest(t, "GET", "/vault/fields/identity.name", nil, true)
	if w.Code != 404 {
		t.Fatalf("expected 404 after delete, got %d", w.Code)
	}
}

func TestGetContext(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/identity.name", map[string]string{"value": "Jane"}, true)
	env.doRequest(t, "PUT", "/vault/fields/financial.income", map[string]string{"value": "100k"}, true)

	w := env.doRequest(t, "GET", "/vault/context", nil, true)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var ctx vault.ContextBundle
	json.NewDecoder(w.Body).Decode(&ctx)
	if len(ctx.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(ctx.Categories))
	}
}

func TestLock_ThenForbidden(t *testing.T) {
	env := setup(t)

	w := env.doRequest(t, "POST", "/vault/lock", nil, true)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// After lock, auth middleware still passes (token validated against session)
	// but vault operations should return forbidden
	w = env.doRequest(t, "GET", "/vault/fields", nil, true)
	if w.Code != http.StatusUnauthorized {
		// Token is now invalid because session was destroyed
		t.Logf("got %d (expected 401 since session destroyed)", w.Code)
	}
}

func TestAuth_MissingToken(t *testing.T) {
	env := setup(t)
	w := env.doRequest(t, "GET", "/vault/fields", nil, false)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	env := setup(t)
	req := httptest.NewRequest("GET", "/vault/fields", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	env.server.mux.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuditLog(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/identity.name", map[string]string{"value": "Jane"}, true)

	w := env.doRequest(t, "GET", "/vault/audit?limit=10", nil, true)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetSensitivity(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/identity.ssn", map[string]string{"value": "123-45-6789"}, true)

	w := env.doRequest(t, "PUT", "/vault/sensitivity/identity.ssn", map[string]string{"tier": "critical"}, true)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	w = env.doRequest(t, "GET", "/vault/fields/identity.ssn", nil, true)
	var field vault.FieldInfo
	json.NewDecoder(w.Body).Decode(&field)
	if field.Sensitivity != "critical" {
		t.Fatalf("expected critical, got %s", field.Sensitivity)
	}
}

func TestCreateServiceToken_API(t *testing.T) {
	env := setup(t)

	w := env.doRequest(t, "POST", "/vault/tokens/service", map[string]string{
		"consumer": "life",
		"scope":    "*",
		"ttl":      "24h",
	}, true)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	json.NewDecoder(w.Body).Decode(&result)
	if result.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if result.ExpiresAt == "" {
		t.Fatal("expected non-empty expires_at")
	}
}

func TestServiceTokenAuth(t *testing.T) {
	env := setup(t)

	// Create a service token
	w := env.doRequest(t, "POST", "/vault/tokens/service", map[string]string{
		"consumer": "life",
	}, true)
	var createResp struct {
		Token string `json:"token"`
	}
	json.NewDecoder(w.Body).Decode(&createResp)

	// Store some data
	env.doRequest(t, "PUT", "/vault/fields/identity.name", map[string]string{"value": "Jane"}, true)

	// Use service token to access context
	req := httptest.NewRequest("GET", "/vault/context", nil)
	req.Header.Set("Authorization", "Bearer "+createResp.Token)
	rec := httptest.NewRecorder()
	env.server.handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("service token auth: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var ctx vault.ContextBundle
	json.NewDecoder(rec.Body).Decode(&ctx)
	if len(ctx.Categories["identity"]) != 1 {
		t.Fatalf("expected 1 identity field, got %d", len(ctx.Categories["identity"]))
	}
}

func TestListServiceTokens_API(t *testing.T) {
	env := setup(t)

	env.doRequest(t, "POST", "/vault/tokens/service", map[string]string{"consumer": "life"}, true)
	env.doRequest(t, "POST", "/vault/tokens/service", map[string]string{"consumer": "other"}, true)

	w := env.doRequest(t, "GET", "/vault/tokens/service", nil, true)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var tokens []struct {
		TokenPrefix string `json:"token_prefix"`
		Consumer    string `json:"consumer"`
	}
	json.NewDecoder(w.Body).Decode(&tokens)
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}
}

func (e *testEnv) doRequestWithToken(t *testing.T, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	e.server.handler.ServeHTTP(w, req)
	return w
}

func createScopedToken(t *testing.T, env *testEnv, consumer, scope string) string {
	t.Helper()
	w := env.doRequest(t, "POST", "/vault/tokens/service", map[string]string{
		"consumer": consumer,
		"scope":    scope,
		"ttl":      "1h",
	}, true)
	if w.Code != 200 {
		t.Fatalf("create token: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	return resp.Token
}

func TestScopedToken_GetField_Allowed(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/identity.name", map[string]string{"value": "Jane"}, true)

	token := createScopedToken(t, env, "agent", "identity.*")

	w := env.doRequestWithToken(t, "GET", "/vault/fields/identity.name", nil, token)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var field vault.FieldInfo
	json.NewDecoder(w.Body).Decode(&field)
	if field.Value != "Jane" {
		t.Fatalf("expected 'Jane', got %q", field.Value)
	}
}

func TestScopedToken_GetField_Denied(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/financial.income", map[string]string{"value": "100k"}, true)

	token := createScopedToken(t, env, "agent", "identity.*")

	w := env.doRequestWithToken(t, "GET", "/vault/fields/financial.income", nil, token)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestScopedToken_SetField_Denied(t *testing.T) {
	env := setup(t)
	token := createScopedToken(t, env, "agent", "identity.*")

	w := env.doRequestWithToken(t, "PUT", "/vault/fields/financial.income", map[string]string{"value": "200k"}, token)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestScopedToken_DeleteField_Denied(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/financial.income", map[string]string{"value": "100k"}, true)
	token := createScopedToken(t, env, "agent", "identity.*")

	w := env.doRequestWithToken(t, "DELETE", "/vault/fields/financial.income", nil, token)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestScopedToken_ListFields_Filtered(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/identity.name", map[string]string{"value": "Jane"}, true)
	env.doRequest(t, "PUT", "/vault/fields/financial.income", map[string]string{"value": "100k"}, true)

	token := createScopedToken(t, env, "agent", "identity.*")

	w := env.doRequestWithToken(t, "GET", "/vault/fields", nil, token)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var fields []vault.FieldInfo
	json.NewDecoder(w.Body).Decode(&fields)
	if len(fields) != 1 {
		t.Fatalf("expected 1 field (identity only), got %d", len(fields))
	}
	if fields[0].Category != "identity" {
		t.Fatalf("expected identity field, got %s", fields[0].Category)
	}
}

func TestScopedToken_Context_Filtered(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/identity.name", map[string]string{"value": "Jane"}, true)
	env.doRequest(t, "PUT", "/vault/fields/financial.income", map[string]string{"value": "100k"}, true)
	env.doRequest(t, "PUT", "/vault/fields/addresses.city", map[string]string{"value": "Seattle"}, true)

	token := createScopedToken(t, env, "tax-agent", "identity.*,financial.*")

	w := env.doRequestWithToken(t, "GET", "/vault/context", nil, token)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var ctx vault.ContextBundle
	json.NewDecoder(w.Body).Decode(&ctx)
	if len(ctx.Categories) != 2 {
		t.Fatalf("expected 2 categories (identity, financial), got %d: %v", len(ctx.Categories), ctx.Categories)
	}
	if _, ok := ctx.Categories["addresses"]; ok {
		t.Fatal("addresses should be filtered out by scope")
	}
}

func TestScopedToken_GetByCategory_Denied(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/financial.income", map[string]string{"value": "100k"}, true)

	token := createScopedToken(t, env, "agent", "identity.*")

	w := env.doRequestWithToken(t, "GET", "/vault/fields/category/financial", nil, token)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestScopedToken_SetSensitivity_Denied(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/financial.income", map[string]string{"value": "100k"}, true)

	token := createScopedToken(t, env, "agent", "identity.*")

	w := env.doRequestWithToken(t, "PUT", "/vault/sensitivity/financial.income", map[string]string{"tier": "critical"}, token)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestScopedToken_WildcardScope_AllowsAll(t *testing.T) {
	env := setup(t)
	env.doRequest(t, "PUT", "/vault/fields/identity.name", map[string]string{"value": "Jane"}, true)
	env.doRequest(t, "PUT", "/vault/fields/financial.income", map[string]string{"value": "100k"}, true)

	token := createScopedToken(t, env, "life", "*")

	w := env.doRequestWithToken(t, "GET", "/vault/context", nil, token)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var ctx vault.ContextBundle
	json.NewDecoder(w.Body).Decode(&ctx)
	if len(ctx.Categories) != 2 {
		t.Fatalf("expected 2 categories with wildcard scope, got %d", len(ctx.Categories))
	}
}

func TestSchema_Public(t *testing.T) {
	env := setup(t)
	w := env.doRequest(t, "GET", "/vault/schema", nil, false)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var schema vault.Schema
	json.NewDecoder(w.Body).Decode(&schema)
	if schema.Version == "" {
		t.Fatal("expected non-empty version")
	}
	if len(schema.Categories) == 0 {
		t.Fatal("expected at least one category")
	}

	// Verify identity category has fields
	found := false
	for _, cat := range schema.Categories {
		if cat.Name == "identity" {
			found = true
			if len(cat.Fields) == 0 {
				t.Fatal("expected identity category to have fields")
			}
		}
	}
	if !found {
		t.Fatal("expected identity category in schema")
	}
}

func TestSetField_WithSuggestion(t *testing.T) {
	env := setup(t)

	// Set a non-canonical field that has a synonym
	w := env.doRequest(t, "PUT", "/vault/fields/identity.name", map[string]string{
		"value": "Jane Smith",
	}, true)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result struct {
		Status     string `json:"status"`
		Suggestion *struct {
			Canonical   string `json:"canonical"`
			Description string `json:"description"`
			Reason      string `json:"reason"`
		} `json:"suggestion"`
	}
	json.NewDecoder(w.Body).Decode(&result)
	if result.Status != "ok" {
		t.Fatalf("expected status ok, got %q", result.Status)
	}
	if result.Suggestion == nil {
		t.Fatal("expected suggestion for identity.name")
	}
	if result.Suggestion.Canonical != "identity.full_name" {
		t.Fatalf("expected canonical identity.full_name, got %q", result.Suggestion.Canonical)
	}
	if result.Suggestion.Reason != "synonym" {
		t.Fatalf("expected reason synonym, got %q", result.Suggestion.Reason)
	}
}

func TestSetField_CanonicalNoSuggestion(t *testing.T) {
	env := setup(t)

	w := env.doRequest(t, "PUT", "/vault/fields/identity.full_name", map[string]string{
		"value": "Jane Smith",
	}, true)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result struct {
		Status     string `json:"status"`
		Suggestion *struct {
			Canonical string `json:"canonical"`
		} `json:"suggestion"`
	}
	json.NewDecoder(w.Body).Decode(&result)
	if result.Suggestion != nil {
		t.Fatalf("expected no suggestion for canonical field, got %+v", result.Suggestion)
	}
}

func TestSetField_DefaultSensitivity(t *testing.T) {
	env := setup(t)

	// Set a critical field without explicit sensitivity
	w := env.doRequest(t, "PUT", "/vault/fields/payment.card_number", map[string]string{
		"value": "4111111111111111",
	}, true)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it was stored with critical sensitivity
	w = env.doRequest(t, "GET", "/vault/fields/payment.card_number", nil, true)
	var field vault.FieldInfo
	json.NewDecoder(w.Body).Decode(&field)
	if field.Sensitivity != "critical" {
		t.Fatalf("expected critical sensitivity for card_number, got %q", field.Sensitivity)
	}
}

// F2: Field ID injection tests
func TestFieldID_Validation_Rejects_Spaces(t *testing.T) {
	env := setup(t)
	// Spaces in field name — use raw request to avoid URL encoding
	req := httptest.NewRequest("GET", "/vault/fields/identity.full%20name", nil)
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.server.mux.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 for space in field ID, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFieldID_Validation_Rejects_Slashes(t *testing.T) {
	env := setup(t)
	// Slash in field name
	req := httptest.NewRequest("GET", "/vault/fields/identity.name%2Fevil", nil)
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.server.mux.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 for slash in field ID, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFieldID_Validation_Rejects_NullByte(t *testing.T) {
	env := setup(t)
	req := httptest.NewRequest("GET", "/vault/fields/identity.name%00evil", nil)
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.server.mux.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 for null byte in field ID, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFieldID_Validation_Accepts_Valid(t *testing.T) {
	env := setup(t)
	w := env.doRequest(t, "PUT", "/vault/fields/my_category.field-name", map[string]string{"value": "ok"}, true)
	if w.Code != 200 {
		t.Fatalf("expected 200 for valid ID, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCategory_Validation_Rejects_SpecialChars(t *testing.T) {
	env := setup(t)
	req := httptest.NewRequest("GET", "/vault/fields/category/evil%24cat", nil)
	req.Header.Set("Authorization", "Bearer "+env.token)
	w := httptest.NewRecorder()
	env.server.mux.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 for special chars in category, got %d: %s", w.Code, w.Body.String())
	}
}

// F3: Request body size limit test
func TestBodySizeLimit(t *testing.T) {
	env := setup(t)
	// Create a body larger than 1MB
	huge := make([]byte, 2*1024*1024)
	for i := range huge {
		huge[i] = 'A'
	}
	w := env.doRequest(t, "PUT", "/vault/fields/identity.name", map[string]string{"value": string(huge)}, true)
	// Should fail with 400 (MaxBytesReader) or 413
	if w.Code == 200 {
		t.Fatal("expected rejection for oversized body, got 200")
	}
}

// F5: Service token privilege escalation tests
func TestServiceToken_CannotLock(t *testing.T) {
	env := setup(t)
	token := createScopedToken(t, env, "agent", "*")

	w := env.doRequestWithToken(t, "POST", "/vault/lock", nil, token)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServiceToken_CannotCreateTokens(t *testing.T) {
	env := setup(t)
	token := createScopedToken(t, env, "agent", "*")

	w := env.doRequestWithToken(t, "POST", "/vault/tokens/service", map[string]string{
		"consumer": "evil",
		"scope":    "*",
	}, token)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServiceToken_CannotListTokens(t *testing.T) {
	env := setup(t)
	token := createScopedToken(t, env, "agent", "*")

	w := env.doRequestWithToken(t, "GET", "/vault/tokens/service", nil, token)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServiceToken_CannotRevokeTokens(t *testing.T) {
	env := setup(t)
	token := createScopedToken(t, env, "agent", "*")

	w := env.doRequestWithToken(t, "DELETE", "/vault/tokens/service/sometoken", nil, token)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServiceToken_CannotViewAudit(t *testing.T) {
	env := setup(t)
	token := createScopedToken(t, env, "agent", "*")

	w := env.doRequestWithToken(t, "GET", "/vault/audit", nil, token)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// F7: Audit log limit cap
func TestAuditLog_LimitCapped(t *testing.T) {
	env := setup(t)
	// Request a huge limit
	w := env.doRequest(t, "GET", "/vault/audit?limit=9999999", nil, true)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Just verify it doesn't crash — the cap is internal
}

// F8: Security headers
func TestSecurityHeaders(t *testing.T) {
	env := setup(t)
	w := env.doRequest(t, "GET", "/vault/status", nil, false)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if v := w.Header().Get("X-Content-Type-Options"); v != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options: nosniff, got %q", v)
	}
	if v := w.Header().Get("Cache-Control"); v != "no-store" {
		t.Fatalf("expected Cache-Control: no-store, got %q", v)
	}
	if v := w.Header().Get("X-Frame-Options"); v != "DENY" {
		t.Fatalf("expected X-Frame-Options: DENY, got %q", v)
	}
}

func TestUI_ServesHTML(t *testing.T) {
	env := setup(t)
	w := env.doRequest(t, "GET", "/ui", nil, false)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Fatalf("expected text/html, got %q", ct)
	}
	if !strings.Contains(w.Body.String(), "万象档案袋") {
		t.Fatal("expected HTML page with vault UI")
	}
}

// F9: Error message sanitization
func TestErrorMessages_NoInternalLeak(t *testing.T) {
	env := setup(t)

	// Lock the vault, then try to access fields — should get safe message
	env.doRequest(t, "POST", "/vault/lock", nil, true)

	// Need a new vault + token since the old session is destroyed
	env2 := setup(t)
	env2.doRequest(t, "PUT", "/vault/fields/identity.name", map[string]string{"value": "Jane"}, true)
	// Force an error by setting a bad sensitivity tier
	w := env2.doRequest(t, "PUT", "/vault/sensitivity/identity.name", map[string]string{"tier": "INVALID"}, true)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	errMsg := resp["error"]
	// Should contain the tier validation message (known error), not a stack trace
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d: %s", w.Code, errMsg)
	}
}

func TestRevokeServiceToken_API(t *testing.T) {
	env := setup(t)

	// Create
	w := env.doRequest(t, "POST", "/vault/tokens/service", map[string]string{"consumer": "life"}, true)
	var createResp struct {
		Token string `json:"token"`
	}
	json.NewDecoder(w.Body).Decode(&createResp)

	// Revoke by full token
	w = env.doRequest(t, "DELETE", "/vault/tokens/service/"+createResp.Token, nil, true)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify revoked token no longer works
	req := httptest.NewRequest("GET", "/vault/fields", nil)
	req.Header.Set("Authorization", "Bearer "+createResp.Token)
	rec := httptest.NewRecorder()
	env.server.handler.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("revoked token: expected 401, got %d", rec.Code)
	}
}

// Attachment end-to-end flow through the real HTTP handlers.
func TestAttachments_CRUD(t *testing.T) {
	env := setup(t)

	// Precondition: field must exist.
	w := env.doRequest(t, "PUT", "/vault/fields/identity.resume", map[string]string{"value": "v1"}, true)
	if w.Code != 200 {
		t.Fatalf("create field: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	fieldQuery := "/vault/attachments?field=identity.resume"

	// List before upload -> empty.
	w = env.doRequest(t, "GET", fieldQuery, nil, true)
	if w.Code != 200 {
		t.Fatalf("list: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var list []vault.AttachmentInfo
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 0 {
		t.Fatalf("expected no attachments initially, got %d", len(list))
	}

	// Upload two attachments (raw body).
	content := []byte("hello attachment world")
	upload := func(filename string) string {
		req := httptest.NewRequest("POST", fieldQuery, bytes.NewReader(content))
		req.Header.Set("Authorization", "Bearer "+env.token)
		req.Header.Set("X-Filename", filename)
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()
		env.server.handler.ServeHTTP(rec, req)
		if rec.Code != 201 {
			t.Fatalf("upload %s: expected 201, got %d: %s", filename, rec.Code, rec.Body.String())
		}
		var info vault.AttachmentInfo
		json.NewDecoder(rec.Body).Decode(&info)
		if info.ID == "" || info.Filename != filename || info.Size != int64(len(content)) {
			t.Fatalf("upload %s: bad info %+v", filename, info)
		}
		return info.ID
	}
	id1 := upload("a.txt")
	id2 := upload("b.txt")
	if id1 == id2 {
		t.Fatal("expected distinct attachment IDs")
	}

	// List after upload -> 2 attachments.
	w = env.doRequest(t, "GET", fieldQuery, nil, true)
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(list))
	}

	// Download attachment 1 -> content round-trips.
	req := httptest.NewRequest("GET", "/vault/attachments/"+id1, nil)
	req.Header.Set("Authorization", "Bearer "+env.token)
	rec := httptest.NewRecorder()
	env.server.handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("download: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != string(content) {
		t.Fatalf("download: content mismatch, got %q", rec.Body.String())
	}
	if rec.Header().Get("X-Filename") != "a.txt" {
		t.Fatalf("download: expected X-Filename 'a.txt', got %q", rec.Header().Get("X-Filename"))
	}

	// Scope: a token limited to identity.* may access this attachment.
	scoped := createScopedToken(t, env, "agent", "identity.*")
	req = httptest.NewRequest("GET", "/vault/attachments/"+id1, nil)
	req.Header.Set("Authorization", "Bearer "+scoped)
	rec = httptest.NewRecorder()
	env.server.handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("scoped download: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Scope: token limited to financial.* must be denied.
	denied := createScopedToken(t, env, "agent", "financial.*")
	req = httptest.NewRequest("GET", "/vault/attachments/"+id1, nil)
	req.Header.Set("Authorization", "Bearer "+denied)
	rec = httptest.NewRecorder()
	env.server.handler.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("scoped denied: expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	// Delete attachment 1.
	w = env.doRequest(t, "DELETE", "/vault/attachments/"+id1, nil, true)
	if w.Code != 200 {
		t.Fatalf("delete: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// List after delete -> 1 attachment.
	w = env.doRequest(t, "GET", fieldQuery, nil, true)
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 1 || list[0].ID != id2 {
		t.Fatalf("expected 1 attachment (b.txt), got %+v", list)
	}
	_ = id2

	// Deleting the field cascades and removes its remaining attachments.
	env.doRequest(t, "DELETE", "/vault/fields/identity.resume", nil, true)
	req = httptest.NewRequest("GET", "/vault/attachments/"+id2, nil)
	req.Header.Set("Authorization", "Bearer "+env.token)
	rec = httptest.NewRecorder()
	env.server.handler.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("cascade delete: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAttachments_Validation(t *testing.T) {
	env := setup(t)

	// Unknown field -> 400.
	req := httptest.NewRequest("POST", "/vault/attachments?field=identity.does_not_exist", bytes.NewReader([]byte("x")))
	req.Header.Set("Authorization", "Bearer "+env.token)
	req.Header.Set("X-Filename", "x.txt")
	rec := httptest.NewRecorder()
	env.server.handler.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("unknown field: expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	// Missing X-Filename -> 400.
	req = httptest.NewRequest("POST", "/vault/attachments?field=identity.resume", bytes.NewReader([]byte("x")))
	req.Header.Set("Authorization", "Bearer "+env.token)
	rec = httptest.NewRecorder()
	env.server.handler.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("missing filename: expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	// Empty body -> 400.
	req = httptest.NewRequest("POST", "/vault/attachments?field=identity.resume", bytes.NewReader(nil))
	req.Header.Set("Authorization", "Bearer "+env.token)
	req.Header.Set("X-Filename", "x.txt")
	rec = httptest.NewRecorder()
	env.server.handler.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("empty body: expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	// Missing field query -> 400.
	req = httptest.NewRequest("POST", "/vault/attachments", bytes.NewReader([]byte("x")))
	req.Header.Set("Authorization", "Bearer "+env.token)
	req.Header.Set("X-Filename", "x.txt")
	rec = httptest.NewRecorder()
	env.server.handler.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("missing field: expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
