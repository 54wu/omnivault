package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
)

// responseRecorder captures the status and body written by an http.Handler.
type responseRecorder struct {
	status int
	header http.Header
	body   *bytes.Buffer
}

func (r *responseRecorder) Header() http.Header         { return r.header }
func (r *responseRecorder) Write(b []byte) (int, error) { return r.body.Write(b) }
func (r *responseRecorder) WriteHeader(code int)        { r.status = code }

// NativeBridge exposes the vault HTTP handler to the in-process UI via
// WebView2 Bind. Every request is executed against the handler in-process,
// so no TCP port or named pipe is involved — nothing can be intercepted by
// network-filter drivers.
type NativeBridge struct {
	handler http.Handler
	token   string
}

// NewNativeBridge builds a bridge that always authenticates as the given
// session token (the vault is already unlocked in-process).
func NewNativeBridge(h http.Handler, token string) *NativeBridge {
	return &NativeBridge{handler: h, token: token}
}

// SetToken adopts a new session token. Used by the in-window login flow: the
// bridge starts with an empty token and is given the real one after the user
// unlocks the vault.
func (b *NativeBridge) SetToken(token string) {
	b.token = token
}

// Request runs one vault API call in-process and returns a JSON envelope:
//
//	{"status":int,"content_type":string,"filename":string,"data_b64":string}
//
// The JSON envelope is returned as a string so it can cross the WebView2 graph
// boundary. data_b64 holds the raw response bytes (JSON text or attachment
// content) base64-encoded.
func (b *NativeBridge) Request(method, path, headersJSON, bodyB64 string) string {
	result := map[string]any{
		"status":       500,
		"content_type": "text/plain",
		"filename":     "",
		"data_b64":     base64.StdEncoding.EncodeToString([]byte("internal error")),
	}
	defer func() {
		if r := recover(); r != nil {
			result["status"] = 500
			result["content_type"] = "text/plain"
			result["filename"] = ""
			result["data_b64"] = base64.StdEncoding.EncodeToString([]byte("internal error"))
		}
	}()

	var headers map[string]string
	json.Unmarshal([]byte(headersJSON), &headers)

	var body []byte
	if bodyB64 != "" {
		if decoded, err := base64.StdEncoding.DecodeString(bodyB64); err == nil {
			body = decoded
		}
	}

	req, err := http.NewRequest(method, "http://omnivault"+path, bytes.NewReader(body))
	if err != nil {
		result["status"] = 400
		result["data_b64"] = base64.StdEncoding.EncodeToString([]byte(err.Error()))
		return encodeBridgeResult(result)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	// Authenticate as the in-process session; the JS layer's token variable is
	// irrelevant here because the vault is already unlocked with this token.
	req.Header.Set("Authorization", "Bearer "+b.token)

	rec := &responseRecorder{status: 200, header: make(http.Header), body: &bytes.Buffer{}}
	b.handler.ServeHTTP(rec, req)

	result["status"] = rec.status
	result["content_type"] = rec.header.Get("Content-Type")
	result["filename"] = rec.header.Get("X-Filename")
	result["data_b64"] = base64.StdEncoding.EncodeToString(rec.body.Bytes())

	// A successful password change invalidates the old session; adopt the new
	// token so subsequent requests keep authenticating.
	if rec.status == http.StatusOK && path == "/vault/change-password" {
		var pwResp struct {
			Token string `json:"token"`
		}
		if json.Unmarshal(rec.body.Bytes(), &pwResp) == nil && pwResp.Token != "" {
			b.token = pwResp.Token
		}
	}

	return encodeBridgeResult(result)
}

func encodeBridgeResult(result map[string]any) string {
	out, _ := json.Marshal(result)
	return string(out)
}

// Handler exposes the full HTTP handler chain so the UI can run it in-process.
func (s *Server) Handler() http.Handler {
	return s.handler
}

// UISource returns the embedded onboarding HTML for the in-process UI.
func UISource() []byte {
	return onboardingHTML
}