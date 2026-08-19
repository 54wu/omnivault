package vault

import (
	crand "crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/54wu/omnivault/internal/crypto"
	"github.com/54wu/omnivault/internal/store"
)

var (
	ErrLocked         = errors.New("vault is locked")
	ErrAlreadyUnlocked = errors.New("vault is already unlocked")
	ErrNotInitialized = errors.New("vault is not initialized")
	ErrAlreadyInit    = errors.New("vault is already initialized")
	ErrWrongPassword  = errors.New("wrong password or secret key")
	ErrInvalidTier    = errors.New("invalid sensitivity tier: must be public, standard, sensitive, or critical")
)

var validTiers = map[string]bool{
	"public": true, "standard": true, "sensitive": true, "critical": true,
}

// Vault is the main entry point for vault operations.
type Vault struct {
	mu      sync.RWMutex
	db      *store.DB
	session *Session
	dir     string // ~/.omnivault
	salt    []byte // loaded on unlock, used for HKDF subkey derivation
	backup  *BackupManager
}

// Open opens an existing vault database.
func Open(dir string) (*Vault, error) {
	dbPath := filepath.Join(dir, "vault.db")
	db, err := store.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	backup := NewBackupManager(dir, defaultBackupKeep, db.Checkpoint)

	// If this open upgraded the schema, take an immediate snapshot so a
	// migrated vault always has a fresh recoverable point before any writes.
	// A snapshot failure must not lock the user out of an already-migrated
	// vault, so it is tolerated rather than treated as fatal.
	if db.Migrated() {
		_ = backup.Snapshot()
	}

	return &Vault{db: db, dir: dir, backup: backup}, nil
}

// Backups lists available versioned backup snapshots, newest first.
func (v *Vault) Backups() ([]BackupInfo, error) {
	return v.backup.List()
}

// BackupNow forces an immediate versioned backup.
func (v *Vault) BackupNow() error {
	return v.backup.Snapshot()
}

// ScheduleBackup coalesces a pending backup after the debounce window.
func (v *Vault) ScheduleBackup() {
	v.backup.Schedule()
}

// StopBackup cancels any pending scheduled backup.
func (v *Vault) StopBackup() {
	v.backup.Stop()
}

// Rollback restores the database from the named backup snapshot. The caller
// must ensure the server is stopped before invoking this.
func (v *Vault) Rollback(name string) error {
	src, err := v.backup.Resolve(name)
	if err != nil {
		return err
	}
	return v.backup.RollbackFile(src)
}

// Init creates a new vault: generates salt, secret key, and stores verification ciphertext.
func Init(dir, password string) (secretKey string, err error) {
	// Create directory
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create vault dir: %w", err)
	}
	dbPath := filepath.Join(dir, "vault.db")
	if _, err := os.Stat(dbPath); err == nil {
		return "", ErrAlreadyInit
	}

	db, err := store.Open(dbPath)
	if err != nil {
		return "", fmt.Errorf("create database: %w", err)
	}
	defer db.Close()

	// Generate salt
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	// Generate secret key
	sk, err := crypto.GenerateSecretKey()
	if err != nil {
		return "", fmt.Errorf("generate secret key: %w", err)
	}

	// Store salt and secret key hash in DB
	if err := db.SetMeta("salt", base64.StdEncoding.EncodeToString(salt)); err != nil {
		return "", err
	}
	if err := db.SetMeta("secret_key_hash", hex.EncodeToString(crypto.HashSecretKey(sk))); err != nil {
		return "", err
	}

	// Derive vault key and create verification ciphertext
	vaultKey := crypto.DeriveVaultKey([]byte(password), sk, salt)
	verifyPlaintext := []byte("omnivault-verification")
	verifyCipher, err := crypto.EncryptToBase64(vaultKey, verifyPlaintext)
	if err != nil {
		return "", fmt.Errorf("create verification: %w", err)
	}
	if err := db.SetMeta("verification", verifyCipher); err != nil {
		return "", err
	}

	skHex := hex.EncodeToString(sk)

	// Write the secret key to its resolved location (external path via
	// OVAULT_KEY_PATH / DPAPI, else the legacy in-vault spot). Keeping it out of
	// the vault folder when an external path is configured preserves the "key
	// stays on device" separation model.
	skPath := SecretKeyPath(dir)
	if pdir := filepath.Dir(skPath); pdir != dir {
		if err := os.MkdirAll(pdir, 0700); err != nil {
			return "", fmt.Errorf("create key dir: %w", err)
		}
	}
	if err := os.WriteFile(skPath, []byte(skHex+"\n"), 0600); err != nil {
		return "", fmt.Errorf("write secret key: %w", err)
	}

	// Zero vault key
	for i := range vaultKey {
		vaultKey[i] = 0
	}

	return skHex, nil
}

// Unlock derives the vault key and creates a session.
func (v *Vault) Unlock(password string, secretKeyHex string) (token string, err error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.session != nil {
		return "", ErrAlreadyUnlocked
	}

	init, err := v.db.IsInitialized()
	if err != nil {
		return "", err
	}
	if !init {
		return "", ErrNotInitialized
	}

	// Load salt
	saltB64, err := v.db.GetMeta("salt")
	if err != nil {
		return "", err
	}
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return "", fmt.Errorf("decode salt: %w", err)
	}

	// Decode secret key
	sk, err := hex.DecodeString(strings.TrimSpace(secretKeyHex))
	if err != nil {
		return "", fmt.Errorf("decode secret key: %w", err)
	}

	// Verify secret key hash
	storedHash, err := v.db.GetMeta("secret_key_hash")
	if err != nil {
		return "", err
	}
	actualHash := hex.EncodeToString(crypto.HashSecretKey(sk))
	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(actualHash)) != 1 {
		return "", ErrWrongPassword
	}

	// Derive vault key
	vaultKey := crypto.DeriveVaultKey([]byte(password), sk, salt)

	// Verify with stored ciphertext
	verifyCipher, err := v.db.GetMeta("verification")
	if err != nil {
		return "", err
	}
	plaintext, err := crypto.DecryptFromBase64(vaultKey, verifyCipher)
	if err != nil {
		return "", ErrWrongPassword
	}
	if string(plaintext) != "omnivault-verification" {
		return "", ErrWrongPassword
	}

	// Store salt for HKDF subkey derivation
	v.salt = salt

	// Create session
	session, err := NewSession(vaultKey, func() {
		v.mu.Lock()
		v.session = nil
		v.mu.Unlock()
	})
	if err != nil {
		return "", err
	}
	v.session = session

	// Zero local copy of vault key
	for i := range vaultKey {
		vaultKey[i] = 0
	}

	// Log access
	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: "*", Action: "unlock"})

	return session.Token(), nil
}

// Lock destroys the session and zeroes the vault key.
func (v *Vault) Lock() {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.session != nil {
		v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: "*", Action: "lock"})
		v.session.Destroy()
		v.session = nil
	}
}

// SetSecurityQuestion stores a security question and a salted, password-derived
// hash of its answer. Used as a recovery gate after repeated wrong passwords.
// Requires the vault to be unlocked so a DB thief cannot install their own question.
func (v *Vault) SetSecurityQuestion(question, answer string) error {
	if _, err := v.requireUnlocked(); err != nil {
		return err
	}
	q := strings.TrimSpace(question)
	a := normalizeAnswer(answer)
	if q == "" || a == "" {
		return errors.New("question and answer required")
	}
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}
	hash := crypto.DeriveVaultKey([]byte(a), nil, salt)
	if err := v.db.SetMeta("sec_question", q); err != nil {
		return err
	}
	if err := v.db.SetMeta("sec_answer_salt", base64.StdEncoding.EncodeToString(salt)); err != nil {
		return err
	}
	if err := v.db.SetMeta("sec_answer_hash", hex.EncodeToString(hash)); err != nil {
		return err
	}
	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: "*", Action: "set_security_question"})
	return nil
}

// HasSecurityQuestion reports whether a security question is configured.
func (v *Vault) HasSecurityQuestion() (bool, error) {
	q, err := v.db.GetMeta("sec_question")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(q) != "", nil
}

// SecurityQuestion returns the configured security question text.
func (v *Vault) SecurityQuestion() (string, error) {
	return v.db.GetMeta("sec_question")
}

// VerifySecurityAnswer checks an answer against the stored hash in constant time.
func (v *Vault) VerifySecurityAnswer(answer string) (bool, error) {
	has, err := v.HasSecurityQuestion()
	if err != nil || !has {
		return false, err
	}
	saltB64, err := v.db.GetMeta("sec_answer_salt")
	if err != nil {
		return false, err
	}
	storedHash, err := v.db.GetMeta("sec_answer_hash")
	if err != nil {
		return false, err
	}
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return false, fmt.Errorf("decode salt: %w", err)
	}
	actual := crypto.DeriveVaultKey([]byte(normalizeAnswer(answer)), nil, salt)
	return subtle.ConstantTimeCompare([]byte(hex.EncodeToString(actual)), []byte(storedHash)) == 1, nil
}

// normalizeAnswer trims and lower-cases a security answer so it is forgiving to
// whitespace and casing differences.
func normalizeAnswer(answer string) string {
	return strings.ToLower(strings.TrimSpace(answer))
}

// ChangePassword re-encrypts every field with a new vault key derived from the
// new profile password, then re-establishes the session. The old session token
// is invalidated; the returned token is the new one.
func (v *Vault) ChangePassword(oldPassword, newPassword string) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.session == nil {
		return "", ErrLocked
	}
	if len(newPassword) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}

	saltB64, err := v.db.GetMeta("salt")
	if err != nil {
		return "", err
	}
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return "", fmt.Errorf("decode salt: %w", err)
	}

	// The secret key is needed to re-derive the vault key; read it from disk
	// and verify it matches the stored hash.
	sk, err := v.loadSecretKey()
	if err != nil {
		return "", err
	}

	// Verify the old password via the stored verification ciphertext.
	oldVaultKey := crypto.DeriveVaultKey([]byte(oldPassword), sk, salt)
	defer func() {
		for i := range oldVaultKey {
			oldVaultKey[i] = 0
		}
	}()
	verifyCipher, err := v.db.GetMeta("verification")
	if err != nil {
		return "", err
	}
	plaintext, err := crypto.DecryptFromBase64(oldVaultKey, verifyCipher)
	if err != nil || string(plaintext) != "omnivault-verification" {
		return "", ErrWrongPassword
	}

	// Derive the new vault key.
	newVaultKey := crypto.DeriveVaultKey([]byte(newPassword), sk, salt)
	defer func() {
		for i := range newVaultKey {
			newVaultKey[i] = 0
		}
	}()

	// Re-encrypt every field with the new category subkeys.
	fields, err := v.db.GetAllFields()
	if err != nil {
		return "", err
	}
	now := time.Now()
	for _, f := range fields {
		oldSub, err := crypto.DeriveSubkey(oldVaultKey, salt, f.Category)
		if err != nil {
			return "", err
		}
		pt, err := crypto.DecryptFromBase64(oldSub, f.Value)
		if err != nil {
			return "", fmt.Errorf("decrypt %s: %w", f.ID, err)
		}
		newSub, err := crypto.DeriveSubkey(newVaultKey, salt, f.Category)
		if err != nil {
			return "", err
		}
		ct, err := crypto.EncryptToBase64(newSub, pt)
		if err != nil {
			return "", err
		}
		if err := v.db.UpdateFieldCipher(f.ID, ct, now); err != nil {
			return "", err
		}
		for i := range oldSub {
			oldSub[i] = 0
		}
		for i := range newSub {
			newSub[i] = 0
		}
	}

	// Re-encrypt every attachment with the new category subkeys.
	atts, err := v.db.GetAllAttachments()
	if err != nil {
		return "", err
	}
	for _, a := range atts {
		oldSub, err := crypto.DeriveSubkey(oldVaultKey, salt, a.Category)
		if err != nil {
			return "", err
		}
		pt, err := crypto.DecryptFromBase64(oldSub, a.Data)
		if err != nil {
			return "", fmt.Errorf("decrypt attachment %s: %w", a.ID, err)
		}
		newSub, err := crypto.DeriveSubkey(newVaultKey, salt, a.Category)
		if err != nil {
			return "", err
		}
		ct, err := crypto.EncryptToBase64(newSub, pt)
		if err != nil {
			return "", err
		}
		if err := v.db.UpdateAttachmentCipher(a.ID, ct); err != nil {
			return "", err
		}
		for i := range oldSub {
			oldSub[i] = 0
		}
		for i := range newSub {
			newSub[i] = 0
		}
	}

	// Re-encrypt the verification ciphertext with the new vault key.
	newVerify, err := crypto.EncryptToBase64(newVaultKey, []byte("omnivault-verification"))
	if err != nil {
		return "", err
	}
	if err := v.db.SetMeta("verification", newVerify); err != nil {
		return "", err
	}

	// Re-establish the session with the new vault key (old session is destroyed).
	v.session.Destroy()
	v.session = nil
	session, err := NewSession(newVaultKey, func() {
		v.mu.Lock()
		v.session = nil
		v.mu.Unlock()
	})
	if err != nil {
		return "", err
	}
	v.session = session

	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: "*", Action: "change_password"})
	return session.Token(), nil
}

// loadSecretKey reads the secret key from disk and verifies it against the
// stored hash. Returns the raw 128-bit key bytes.
func (v *Vault) loadSecretKey() ([]byte, error) {
	path := SecretKeyPath(v.dir)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("secret key not found at %s (change-password needs the key file)", path)
	}
	sk, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(sk) == 0 {
		return nil, errors.New("invalid secret key file")
	}
	storedHash, err := v.db.GetMeta("secret_key_hash")
	if err != nil {
		return nil, err
	}
	actualHash := hex.EncodeToString(crypto.HashSecretKey(sk))
	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(actualHash)) != 1 {
		return nil, errors.New("secret key does not match vault")
	}
	return sk, nil
}

// Status returns the current vault status.
func (v *Vault) Status() (*VaultStatus, error) {
	init, err := v.db.IsInitialized()
	if err != nil {
		return nil, err
	}

	status := &VaultStatus{
		Initialized: init,
		Locked:      true,
	}

	v.mu.RLock()
	if v.session != nil {
		status.Locked = false
	}
	v.mu.RUnlock()

	if init {
		count, _ := v.db.FieldCount()
		status.FieldCount = count
		cats, _ := v.db.CategoryCounts()
		status.Categories = cats
	}

	return status, nil
}

// Set encrypts and stores a field value.
func (v *Vault) Set(id, value, sensitivity string) error {
	if err := ValidateFieldID(id); err != nil {
		return err
	}

	vaultKey, err := v.requireUnlocked()
	if err != nil {
		return err
	}

	parts := strings.SplitN(id, ".", 2)
	category, fieldName := parts[0], parts[1]

	// Derive category subkey
	subkey, err := crypto.DeriveSubkey(vaultKey, v.salt, category)
	if err != nil {
		return fmt.Errorf("derive subkey: %w", err)
	}

	// Encrypt
	encrypted, err := crypto.EncryptToBase64(subkey, []byte(value))
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	if sensitivity == "" {
		sensitivity = "standard"
	}
	if !validTiers[sensitivity] {
		return ErrInvalidTier
	}

	err = v.db.SetField(store.Field{
		ID:          id,
		Category:    category,
		FieldName:   fieldName,
		Value:       encrypted,
		Sensitivity: sensitivity,
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		return err
	}

	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: id, Action: "write"})
	return nil
}

// Get decrypts and returns a field value.
func (v *Vault) Get(id string) (*FieldInfo, error) {
	vaultKey, err := v.requireUnlocked()
	if err != nil {
		return nil, err
	}

	f, err := v.db.GetField(id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, nil
	}

	subkey, err := crypto.DeriveSubkey(vaultKey, v.salt, f.Category)
	if err != nil {
		return nil, err
	}

	plaintext, err := crypto.DecryptFromBase64(subkey, f.Value)
	if err != nil {
		return nil, fmt.Errorf("decrypt field %s: %w", id, err)
	}

	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: id, Action: "read"})

	return &FieldInfo{
		ID:          f.ID,
		Category:    f.Category,
		FieldName:   f.FieldName,
		Value:       string(plaintext),
		Sensitivity: f.Sensitivity,
		UpdatedAt:   f.UpdatedAt,
		Version:     f.Version,
	}, nil
}

// List returns all field metadata (no values).
func (v *Vault) List() ([]FieldInfo, error) {
	if _, err := v.requireUnlocked(); err != nil {
		return nil, err
	}

	fields, err := v.db.ListFields()
	if err != nil {
		return nil, err
	}

	result := make([]FieldInfo, len(fields))
	for i, f := range fields {
		result[i] = FieldInfo{
			ID:          f.ID,
			Category:    f.Category,
			FieldName:   f.FieldName,
			Sensitivity: f.Sensitivity,
			UpdatedAt:   f.UpdatedAt,
			Version:     f.Version,
		}
	}
	return result, nil
}

// ListByCategory returns field metadata for a category (no values).
func (v *Vault) ListByCategory(category string) ([]FieldInfo, error) {
	if _, err := v.requireUnlocked(); err != nil {
		return nil, err
	}

	fields, err := v.db.ListFieldsByCategory(category)
	if err != nil {
		return nil, err
	}

	result := make([]FieldInfo, len(fields))
	for i, f := range fields {
		result[i] = FieldInfo{
			ID:          f.ID,
			Category:    f.Category,
			FieldName:   f.FieldName,
			Sensitivity: f.Sensitivity,
			UpdatedAt:   f.UpdatedAt,
			Version:     f.Version,
		}
	}
	return result, nil
}

// GetByCategory returns all decrypted fields for a category.
func (v *Vault) GetByCategory(category string) ([]FieldInfo, error) {
	vaultKey, err := v.requireUnlocked()
	if err != nil {
		return nil, err
	}

	fields, err := v.db.GetFieldsByCategory(category)
	if err != nil {
		return nil, err
	}

	subkey, err := crypto.DeriveSubkey(vaultKey, v.salt, category)
	if err != nil {
		return nil, err
	}

	result := make([]FieldInfo, len(fields))
	for i, f := range fields {
		plaintext, err := crypto.DecryptFromBase64(subkey, f.Value)
		if err != nil {
			return nil, fmt.Errorf("decrypt %s: %w", f.ID, err)
		}
		result[i] = FieldInfo{
			ID:          f.ID,
			Category:    f.Category,
			FieldName:   f.FieldName,
			Value:       string(plaintext),
			Sensitivity: f.Sensitivity,
			UpdatedAt:   f.UpdatedAt,
			Version:     f.Version,
		}
	}

	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: category + ".*", Action: "read"})
	return result, nil
}

// GetContext returns all decrypted fields grouped by category.
func (v *Vault) GetContext() (*ContextBundle, error) {
	vaultKey, err := v.requireUnlocked()
	if err != nil {
		return nil, err
	}

	fields, err := v.db.GetAllFields()
	if err != nil {
		return nil, err
	}

	bundle := &ContextBundle{Categories: make(map[string][]FieldInfo)}
	subkeys := make(map[string][]byte)

	for _, f := range fields {
		sk, ok := subkeys[f.Category]
		if !ok {
			sk, err = crypto.DeriveSubkey(vaultKey, v.salt, f.Category)
			if err != nil {
				return nil, err
			}
			subkeys[f.Category] = sk
		}

		plaintext, err := crypto.DecryptFromBase64(sk, f.Value)
		if err != nil {
			return nil, fmt.Errorf("decrypt %s: %w", f.ID, err)
		}

		bundle.Categories[f.Category] = append(bundle.Categories[f.Category], FieldInfo{
			ID:          f.ID,
			Category:    f.Category,
			FieldName:   f.FieldName,
			Value:       string(plaintext),
			Sensitivity: f.Sensitivity,
			UpdatedAt:   f.UpdatedAt,
			Version:     f.Version,
		})
	}

	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: "*", Action: "context"})
	return bundle, nil
}

// Delete removes a field and any attachments associated with it.
func (v *Vault) Delete(id string) error {
	if _, err := v.requireUnlocked(); err != nil {
		return err
	}

	if err := v.db.DeleteAttachmentsByField(id); err != nil {
		return err
	}
	if err := v.db.DeleteField(id); err != nil {
		return err
	}

	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: id, Action: "delete"})
	return nil
}

// SetSensitivity updates a field's sensitivity tier.
func (v *Vault) SetSensitivity(id, tier string) error {
	if _, err := v.requireUnlocked(); err != nil {
		return err
	}
	if !validTiers[tier] {
		return ErrInvalidTier
	}
	return v.db.SetSensitivity(id, tier)
}

// AttachmentInfo is a decrypted attachment returned to callers.
type AttachmentInfo struct {
	ID          string    `json:"id"`
	FieldID     string    `json:"field_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
}

// AddAttachment encrypts and stores a file attachment on a field. The field
// must exist and be unlocked. The attachment is encrypted with the field's
// category subkey, keeping it consistent with field encryption.
func (v *Vault) AddAttachment(fieldID, filename, contentType string, data []byte) (*AttachmentInfo, error) {
	vaultKey, err := v.requireUnlocked()
	if err != nil {
		return nil, err
	}

	f, err := v.db.GetField(fieldID)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("field not found: %s", fieldID)
	}

	subkey, err := crypto.DeriveSubkey(vaultKey, v.salt, f.Category)
	if err != nil {
		return nil, fmt.Errorf("derive subkey: %w", err)
	}
	defer func() {
		for i := range subkey {
			subkey[i] = 0
		}
	}()

	encrypted, err := crypto.EncryptToBase64(subkey, data)
	if err != nil {
		return nil, fmt.Errorf("encrypt attachment: %w", err)
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	id := newAttachmentID()
	now := time.Now()
	a := store.Attachment{
		ID:          id,
		FieldID:     fieldID,
		Category:    f.Category,
		Filename:    filename,
		ContentType: contentType,
		Size:        int64(len(data)),
		Data:        encrypted,
		CreatedAt:   now,
	}
	if err := v.db.CreateAttachment(a); err != nil {
		return nil, err
	}

	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: fieldID, Action: "add_attachment"})
	return &AttachmentInfo{
		ID:          id,
		FieldID:     fieldID,
		Filename:    filename,
		ContentType: contentType,
		Size:        int64(len(data)),
		CreatedAt:   now,
	}, nil
}

// ListAttachments returns attachment metadata for a field (no content).
func (v *Vault) ListAttachments(fieldID string) ([]AttachmentInfo, error) {
	if _, err := v.requireUnlocked(); err != nil {
		return nil, err
	}
	metas, err := v.db.ListAttachmentsByField(fieldID)
	if err != nil {
		return nil, err
	}
	result := make([]AttachmentInfo, 0, len(metas))
	for _, m := range metas {
		result = append(result, AttachmentInfo{
			ID:          m.ID,
			FieldID:     m.FieldID,
			Filename:    m.Filename,
			ContentType: m.ContentType,
			Size:        m.Size,
			CreatedAt:   m.CreatedAt,
		})
	}
	return result, nil
}

// GetAttachment decrypts and returns a single attachment's content.
func (v *Vault) GetAttachment(id string) (*AttachmentInfo, []byte, error) {
	vaultKey, err := v.requireUnlocked()
	if err != nil {
		return nil, nil, err
	}
	a, err := v.db.GetAttachment(id)
	if err != nil {
		return nil, nil, err
	}
	if a == nil {
		return nil, nil, nil
	}

	subkey, err := crypto.DeriveSubkey(vaultKey, v.salt, a.Category)
	if err != nil {
		return nil, nil, fmt.Errorf("derive subkey: %w", err)
	}
	defer func() {
		for i := range subkey {
			subkey[i] = 0
		}
	}()

	plaintext, err := crypto.DecryptFromBase64(subkey, a.Data)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypt attachment %s: %w", a.ID, err)
	}

	v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: a.FieldID, Action: "read_attachment"})
	return &AttachmentInfo{
		ID:          a.ID,
		FieldID:     a.FieldID,
		Filename:    a.Filename,
		ContentType: a.ContentType,
		Size:        a.Size,
		CreatedAt:   a.CreatedAt,
	}, plaintext, nil
}

// DeleteAttachment removes an attachment by ID.
func (v *Vault) DeleteAttachment(id string) (bool, error) {
	if _, err := v.requireUnlocked(); err != nil {
		return false, err
	}
	n, err := v.db.DeleteAttachment(id)
	if err != nil {
		return false, err
	}
	if n > 0 {
		v.db.LogAccess(store.AuditEntry{Consumer: "vault", Scope: id, Action: "delete_attachment"})
	}
	return n > 0, nil
}

// AuditLog returns recent audit entries.
func (v *Vault) AuditLog(limit int) ([]store.AuditEntry, error) {
	return v.db.GetAuditLog(limit)
}

// ValidateToken checks a session token.
func (v *Vault) ValidateToken(token string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.session == nil {
		return false
	}
	return v.session.ValidateToken(token)
}

// hashServiceToken returns the hex-encoded SHA-256 hash of a token.
func hashServiceToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// newAttachmentID returns a random hex attachment ID.
func newAttachmentID() string {
	b := make([]byte, 16)
	if _, err := crand.Read(b); err != nil {
		// Fall back to a timestamp-based ID; collisions are practically nil.
		return "att-" + time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b)
}

// CreateServiceToken generates a long-lived service token for a consumer.
// The raw token is returned to the caller; only the SHA-256 hash is stored.
func (v *Vault) CreateServiceToken(consumer, scope string, ttl time.Duration) (string, error) {
	if _, err := v.requireUnlocked(); err != nil {
		return "", err
	}

	tokenBytes := make([]byte, 32)
	if _, err := crand.Read(tokenBytes); err != nil {
		return "", err
	}
	tokenStr := hex.EncodeToString(tokenBytes)

	t := store.Token{
		TokenStr:  hashServiceToken(tokenStr),
		Consumer:  consumer,
		Scope:     scope,
		ExpiresAt: time.Now().Add(ttl),
		Usage:     "service",
		CreatedAt: time.Now(),
	}
	if err := v.db.CreateToken(t); err != nil {
		return "", err
	}

	v.db.LogAccess(store.AuditEntry{
		Consumer: "vault",
		Scope:    scope,
		Action:   "create_service_token",
		Purpose:  "consumer: " + consumer,
	})

	return tokenStr, nil
}

// ValidateServiceToken checks a service token by hashing it and looking up the hash.
func (v *Vault) ValidateServiceToken(token string) (*store.Token, bool) {
	t, err := v.db.GetToken(hashServiceToken(token))
	if err != nil || t == nil {
		return nil, false
	}
	if t.Usage != "service" {
		return nil, false
	}
	return t, true
}

// ListServiceTokens returns all service tokens.
func (v *Vault) ListServiceTokens() ([]store.Token, error) {
	if _, err := v.requireUnlocked(); err != nil {
		return nil, err
	}
	return v.db.ListTokensByUsage("service")
}

// RevokeServiceToken removes a service token by its hash.
func (v *Vault) RevokeServiceToken(token string) (int64, error) {
	if _, err := v.requireUnlocked(); err != nil {
		return 0, err
	}
	n, err := v.db.DeleteToken(hashServiceToken(token))
	if err != nil {
		return 0, err
	}
	if n > 0 {
		v.db.LogAccess(store.AuditEntry{
			Consumer: "vault",
			Scope:    "*",
			Action:   "revoke_service_token",
		})
	}
	return n, nil
}

// TouchSession resets the auto-lock timer.
func (v *Vault) TouchSession() {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.session != nil {
		v.session.Touch()
	}
}

// Close closes the database.
func (v *Vault) Close() error {
	v.Lock()
	return v.db.Close()
}

// LogAccess writes an entry to the audit log.
func (v *Vault) LogAccess(entry store.AuditEntry) {
	v.db.LogAccess(entry)
}

func (v *Vault) requireUnlocked() ([]byte, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.session == nil {
		return nil, ErrLocked
	}
	v.session.Touch()
	key := v.session.VaultKey()
	if key == nil {
		return nil, ErrLocked
	}
	return key, nil
}
