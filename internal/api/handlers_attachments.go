package api

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/54wu/omnivault/internal/vault"
)

// POST /vault/attachments?field={fieldID}
// Body is raw bytes (the file content). Headers: Content-Type = file's MIME,
// X-Filename = original file name.
func (s *Server) handleAddAttachment(w http.ResponseWriter, r *http.Request) {
	fieldID := r.URL.Query().Get("field")
	if err := vault.ValidateFieldID(fieldID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if !vault.ScopeAllows(scopeFromRequest(r), fieldID) {
		scopeDenied(w)
		return
	}

	filename := strings.TrimSpace(r.Header.Get("X-Filename"))
	if filename == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "X-Filename header required")
		return
	}
	if len(filename) > 255 {
		writeError(w, http.StatusBadRequest, "invalid_request", "filename too long")
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "too_large", "attachment too large")
		return
	}
	if len(data) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "empty attachment")
		return
	}

	ctype := r.Header.Get("Content-Type")
	info, err := s.vault.AddAttachment(fieldID, filename, ctype, data)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	s.vault.ScheduleBackup()
	writeJSON(w, http.StatusCreated, info)
}

// GET /vault/attachments?field={fieldID}
func (s *Server) handleListAttachments(w http.ResponseWriter, r *http.Request) {
	fieldID := r.URL.Query().Get("field")
	if err := vault.ValidateFieldID(fieldID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if !vault.ScopeAllows(scopeFromRequest(r), fieldID) {
		scopeDenied(w)
		return
	}
	metas, err := s.vault.ListAttachments(fieldID)
	if err != nil {
		handleVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, metas)
}

// GET /vault/attachments/{id}
func (s *Server) handleGetAttachment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	info, data, err := s.vault.GetAttachment(id)
	if err != nil {
		handleVaultError(w, err)
		return
	}
	if info == nil {
		writeError(w, http.StatusNotFound, "not_found", "attachment not found")
		return
	}
	if !vault.ScopeAllows(scopeFromRequest(r), info.FieldID) {
		scopeDenied(w)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+urlEscape(info.Filename))
	w.Header().Set("X-Filename", info.Filename)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

// DELETE /vault/attachments/{id}
func (s *Server) handleDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Permission check requires the field id; look it up via the vault.
	info, _, err := s.vault.GetAttachment(id)
	if err != nil {
		handleVaultError(w, err)
		return
	}
	if info == nil {
		writeError(w, http.StatusNotFound, "not_found", "attachment not found")
		return
	}
	if !vault.ScopeAllows(scopeFromRequest(r), info.FieldID) {
		scopeDenied(w)
		return
	}

	ok, err := s.vault.DeleteAttachment(id)
	if err != nil {
		handleVaultError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "attachment not found")
		return
	}
	s.vault.ScheduleBackup()
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// urlEscape percent-encodes a string for use in Content-Disposition.
func urlEscape(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0xf])
		}
	}
	return b.String()
}