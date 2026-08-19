package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/54wu/omnivault/internal/vault"
)

// rateLimiter tracks attempts within a time window.
type rateLimiter struct {
	mu       sync.Mutex
	attempts []time.Time
	max      int
	window   time.Duration
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{max: max, window: window}
}

// allow returns true if the request is within the rate limit.
func (rl *rateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Remove expired attempts
	valid := rl.attempts[:0]
	for _, t := range rl.attempts {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	rl.attempts = valid

	if len(rl.attempts) >= rl.max {
		return false
	}
	rl.attempts = append(rl.attempts, now)
	return true
}

// reset clears all tracked attempts, used after a successful security answer
// so the user is not immediately re-blocked by the frequency limiter.
func (rl *rateLimiter) reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.attempts = rl.attempts[:0]
}

// unlockGuard locks out unlock attempts after too many consecutive wrong
// passwords. If a security question is configured, it demands the correct
// answer to continue; otherwise it falls back to a timed lockout.
type unlockGuard struct {
	mu             sync.Mutex
	failures       int
	answerFailures int
	lockedUntil    time.Time
	needsAnswer    bool
	maxFailures    int
	lockout        time.Duration
}

func newUnlockGuard(maxFailures int, lockout time.Duration) *unlockGuard {
	return &unlockGuard{maxFailures: maxFailures, lockout: lockout}
}

// requiresAnswer reports whether a security answer must be supplied to proceed.
func (g *unlockGuard) requiresAnswer() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.needsAnswer
}

// blocked returns a non-nil error (with remaining cooldown) if the guard is
// currently locked out by time.
func (g *unlockGuard) blocked() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.needsAnswer {
		return nil
	}
	if remaining := time.Until(g.lockedUntil); remaining > 0 {
		return fmt.Errorf("解锁尝试次数过多，请在 %s 后重试", remaining.Round(time.Second))
	}
	return nil
}

// failure records a wrong password attempt. When the threshold is reached the
// guard demands a security answer (if configured) or starts a timed lockout.
func (g *unlockGuard) failure(hasQuestion bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.needsAnswer || time.Now().Before(g.lockedUntil) {
		return
	}
	g.failures++
	if g.failures >= g.maxFailures {
		g.failures = 0
		if hasQuestion {
			g.needsAnswer = true
		} else {
			g.lockedUntil = time.Now().Add(g.lockout)
		}
	}
}

func (g *unlockGuard) answerFailed() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.needsAnswer {
		return
	}
	g.answerFailures++
	if g.answerFailures >= 3 {
		// Repeated wrong answers escalate to a timed lockout.
		g.needsAnswer = false
		g.answerFailures = 0
		g.lockedUntil = time.Now().Add(g.lockout)
	}
}

func (g *unlockGuard) answerSucceeded() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.needsAnswer = false
	g.failures = 0
	g.answerFailures = 0
}

func (g *unlockGuard) success() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.failures = 0
	g.answerFailures = 0
}

// Server is the HTTP API server for the vault.
type Server struct {
	vault       *vault.Vault
	mux         *http.ServeMux
	handler     http.Handler // full chain: bodySizeMiddleware → mux
	server      *http.Server
	unlockLimit *rateLimiter
	unlockGuard *unlockGuard

	// localMu guards the optional in-process loopback listener used by the
	// native UI to expose the vault to external MCP/HTTP consumers on demand.
	localMu       sync.Mutex
	localListener net.Listener
}

// New creates a new API server.
func New(v *vault.Vault, addr string) *Server {
	s := &Server{
		vault:       v,
		unlockLimit: newRateLimiter(5, time.Minute),
		unlockGuard: newUnlockGuard(5, 5*time.Minute),
	}
	s.mux = http.NewServeMux()
	s.registerRoutes()
	s.handler = securityHeadersMiddleware(bodySizeMiddleware(s.mux))
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.handler,
	}
	return s
}

// SetVault swaps the vault instance the server operates on. Used by the
// in-window first-run flow, which closes the stale (uninitialized) handle,
// re-initializes the vault, and reopens it before the UI is shown.
func (s *Server) SetVault(v *vault.Vault) {
	s.vault = v
}

func (s *Server) registerRoutes() {
	// Public endpoints (no auth required)
	s.mux.HandleFunc("GET /ui", s.handleUI)
	s.mux.HandleFunc("GET /assets/", s.handleUIAsset)
	s.mux.HandleFunc("GET /ui/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui", http.StatusMovedPermanently)
	})
	s.mux.HandleFunc("POST /vault/unlock", s.handleUnlock)
	s.mux.HandleFunc("GET /vault/status", s.handleStatus)
	s.mux.HandleFunc("GET /vault/schema", s.handleSchema)

	// Protected endpoints
	protected := http.NewServeMux()
	protected.HandleFunc("POST /vault/lock", s.handleLock)
	protected.HandleFunc("POST /vault/security-question", s.handleSetSecurityQuestion)
	protected.HandleFunc("POST /vault/change-password", s.handleChangePassword)
	protected.HandleFunc("GET /vault/fields", s.handleListFields)
	protected.HandleFunc("GET /vault/fields/category/{category}", s.handleGetByCategory)
	protected.HandleFunc("GET /vault/fields/{id...}", s.handleGetField)
	protected.HandleFunc("PUT /vault/fields/{id...}", s.handleSetField)
	protected.HandleFunc("DELETE /vault/fields/{id...}", s.handleDeleteField)
	protected.HandleFunc("GET /vault/context", s.handleGetContext)
	protected.HandleFunc("GET /vault/audit", s.handleAuditLog)
	protected.HandleFunc("PUT /vault/sensitivity/{id...}", s.handleSetSensitivity)
	protected.HandleFunc("POST /vault/attachments", s.handleAddAttachment)
	protected.HandleFunc("GET /vault/attachments", s.handleListAttachments)
	protected.HandleFunc("GET /vault/attachments/{id}", s.handleGetAttachment)
	protected.HandleFunc("DELETE /vault/attachments/{id}", s.handleDeleteAttachment)
	protected.HandleFunc("POST /vault/tokens/service", s.handleCreateServiceToken)
	protected.HandleFunc("GET /vault/tokens/service", s.handleListServiceTokens)
	protected.HandleFunc("DELETE /vault/tokens/service/{token}", s.handleRevokeServiceToken)
	protected.HandleFunc("POST /vault/merge/plan", s.handleMergePlan)
	protected.HandleFunc("POST /vault/merge/apply", s.handleMergeApply)

	s.mux.Handle("/", s.authMiddleware(protected))
}

// Start begins listening. Returns immediately; use the returned listener to get the actual port.
func (s *Server) Start() (net.Listener, error) {
	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return nil, err
	}
	go s.server.Serve(ln)
	return ln, nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// StartLocal begins listening on the given loopback address (e.g. "127.0.0.1:7200")
// in-process so external MCP/HTTP consumers can reach the vault. It returns an
// error immediately if the address is already in use. The listener lives only as
// long as the owning process — closing the UI stops it (关窗即关).
func (s *Server) StartLocal(addr string) error {
	s.localMu.Lock()
	defer s.localMu.Unlock()
	if s.localListener != nil {
		return nil // already running
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.localListener = ln
	go s.server.Serve(ln)
	return nil
}

// StopLocal closes the in-process loopback listener, if any.
func (s *Server) StopLocal() {
	s.localMu.Lock()
	defer s.localMu.Unlock()
	if s.localListener != nil {
		s.localListener.Close()
		s.localListener = nil
	}
}

// LocalRunning reports whether the in-process loopback listener is active.
func (s *Server) LocalRunning() bool {
	s.localMu.Lock()
	defer s.localMu.Unlock()
	return s.localListener != nil
}
