package vault

import (
	"bytes"
	"testing"
)

func TestAttachmentRoundTrip(t *testing.T) {
	v, _ := tmpVault(t)

	// Set a field first (attachments require an existing field).
	if err := v.Set("p1_identity.name", "张三", "standard"); err != nil {
		t.Fatal(err)
	}

	content := []byte("attachment-bytes-123")
	info, err := v.AddAttachment("p1_identity.name", "resume.pdf", "application/pdf", content)
	if err != nil {
		t.Fatal(err)
	}
	if info.Filename != "resume.pdf" || info.Size != int64(len(content)) {
		t.Fatalf("unexpected attachment info: %+v", info)
	}

	// List
	metas, err := v.ListAttachments("p1_identity.name")
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) != 1 || metas[0].ID != info.ID {
		t.Fatalf("expected 1 attachment, got %+v", metas)
	}

	// Get + decrypt round-trip
	gotInfo, gotData, err := v.GetAttachment(info.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotInfo == nil || !bytes.Equal(gotData, content) {
		t.Fatalf("round-trip mismatch: got %q want %q", gotData, content)
	}

	// Change password re-encrypts attachments; verify still readable.
	newPw := "new-password-456"
	newToken, err := v.ChangePassword(testPassword, newPw)
	if err != nil {
		t.Fatal(err)
	}
	if newToken == "" {
		t.Fatal("expected new token")
	}
	_, gotData2, err := v.GetAttachment(info.ID)
	if err != nil {
		t.Fatalf("attachment unreadable after change-password: %v", err)
	}
	if !bytes.Equal(gotData2, content) {
		t.Fatalf("attachment corrupted after change-password: %q", gotData2)
	}

	// Delete
	ok, err := v.DeleteAttachment(info.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected delete to affect a row")
	}
	metas, _ = v.ListAttachments("p1_identity.name")
	if len(metas) != 0 {
		t.Fatalf("expected 0 attachments after delete, got %d", len(metas))
	}
}

func TestAttachmentRequiresExistingField(t *testing.T) {
	v, _ := tmpVault(t)
	if _, err := v.AddAttachment("p1_identity.nonexistent", "x.txt", "", []byte("x")); err == nil {
		t.Fatal("expected error attaching to a nonexistent field")
	}
}