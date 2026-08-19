package api

import (
	_ "embed"
	"net/http"
	"strings"
)

//go:embed ui/onboarding.html
var onboardingHTML []byte

//go:embed ui/sponsor_wechat.jpg
var sponsorWechat []byte

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(onboardingHTML)
}

// handleUIAsset serves non-HTML UI assets (e.g. the embedded WeChat reward QR
// code) from memory, so the UI never needs a network connection.
func (s *Server) handleUIAsset(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/assets/sponsor_wechat"):
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(sponsorWechat)
	default:
		http.NotFound(w, r)
	}
}
