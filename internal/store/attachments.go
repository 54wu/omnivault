package store

import (
	"database/sql"
	"time"
)

// Attachment represents a row in vault_attachments. Data holds the encrypted
// attachment bytes (base64). Other fields are plaintext metadata.
type Attachment struct {
	ID          string
	FieldID     string
	Category    string
	Filename    string
	ContentType string
	Size        int64
	Data        string // encrypted bytes, base64
	CreatedAt   time.Time
}

// AttachmentMeta is attachment metadata without the encrypted content.
type AttachmentMeta struct {
	ID          string
	FieldID     string
	Filename    string
	ContentType string
	Size        int64
	CreatedAt   time.Time
}

// CreateAttachment inserts a new attachment row.
func (d *DB) CreateAttachment(a Attachment) error {
	_, err := d.conn.Exec(
		`INSERT INTO vault_attachments (id, field_id, category, filename, content_type, size, data, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.FieldID, a.Category, a.Filename, a.ContentType, a.Size, a.Data,
		a.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// GetAttachment retrieves a single attachment including encrypted data.
func (d *DB) GetAttachment(id string) (*Attachment, error) {
	var a Attachment
	var created string
	err := d.conn.QueryRow(
		`SELECT id, field_id, category, filename, content_type, size, data, created_at
		 FROM vault_attachments WHERE id = ?`,
		id,
	).Scan(&a.ID, &a.FieldID, &a.Category, &a.Filename, &a.ContentType, &a.Size, &a.Data, &created)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	a.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return &a, nil
}

// ListAttachmentsByField returns attachment metadata for a field (no content).
func (d *DB) ListAttachmentsByField(fieldID string) ([]AttachmentMeta, error) {
	rows, err := d.conn.Query(
		`SELECT id, field_id, filename, content_type, size, created_at
		 FROM vault_attachments WHERE field_id = ? ORDER BY created_at`,
		fieldID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metas []AttachmentMeta
	for rows.Next() {
		var m AttachmentMeta
		var created string
		if err := rows.Scan(&m.ID, &m.FieldID, &m.Filename, &m.ContentType, &m.Size, &created); err != nil {
			return nil, err
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, created)
		metas = append(metas, m)
	}
	return metas, rows.Err()
}

// DeleteAttachment removes an attachment by ID, returning rows affected.
func (d *DB) DeleteAttachment(id string) (int64, error) {
	res, err := d.conn.Exec("DELETE FROM vault_attachments WHERE id = ?", id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// GetAllAttachments returns all attachments including encrypted data. Used for
// re-encryption during a password change.
func (d *DB) GetAllAttachments() ([]Attachment, error) {
	rows, err := d.conn.Query(
		`SELECT id, field_id, category, filename, content_type, size, data, created_at
		 FROM vault_attachments`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var atts []Attachment
	for rows.Next() {
		var a Attachment
		var created string
		if err := rows.Scan(&a.ID, &a.FieldID, &a.Category, &a.Filename, &a.ContentType, &a.Size, &a.Data, &created); err != nil {
			return nil, err
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, created)
		atts = append(atts, a)
	}
	return atts, rows.Err()
}

// UpdateAttachmentCipher rewrites an attachment's encrypted data without
// changing its metadata. Used during a password change.
func (d *DB) UpdateAttachmentCipher(id, ciphertext string) error {
	_, err := d.conn.Exec("UPDATE vault_attachments SET data = ? WHERE id = ?", ciphertext, id)
	return err
}

// DeleteAttachmentsByField removes all attachments for a field.
func (d *DB) DeleteAttachmentsByField(fieldID string) error {
	_, err := d.conn.Exec("DELETE FROM vault_attachments WHERE field_id = ?", fieldID)
	return err
}