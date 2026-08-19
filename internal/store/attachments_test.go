package store

import (
	"testing"
	"time"
)

func TestAttachmentCRUD(t *testing.T) {
	db := tmpDB(t)

	// Create
	a := Attachment{
		ID:          "att1",
		FieldID:     "p1_identity.name",
		Category:    "identity",
		Filename:    "photo.jpg",
		ContentType: "image/jpeg",
		Size:        42,
		Data:        "ZWNyeXB0ZWQ=",
		CreatedAt:   time.Now(),
	}
	if err := db.CreateAttachment(a); err != nil {
		t.Fatal(err)
	}

	// Get
	got, err := db.GetAttachment("att1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected attachment, got nil")
	}
	if got.Filename != "photo.jpg" || got.Data != "ZWNyeXB0ZWQ=" || got.Category != "identity" {
		t.Fatalf("unexpected attachment: %+v", got)
	}

	// List by field
	metas, err := db.ListAttachmentsByField("p1_identity.name")
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) != 1 || metas[0].ID != "att1" {
		t.Fatalf("expected 1 meta, got %+v", metas)
	}

	// Update cipher
	if err := db.UpdateAttachmentCipher("att1", "TkVX"); err != nil {
		t.Fatal(err)
	}
	got2, _ := db.GetAttachment("att1")
	if got2.Data != "TkVX" {
		t.Fatalf("expected updated cipher, got %q", got2.Data)
	}

	// Delete
	n, err := db.DeleteAttachment("att1")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 row deleted, got %d", n)
	}
	if got, _ := db.GetAttachment("att1"); got != nil {
		t.Fatal("expected attachment to be deleted")
	}
}

func TestDeleteAttachmentsByField(t *testing.T) {
	db := tmpDB(t)
	for _, id := range []string{"a1", "a2", "b1"} {
		field := "p1_identity.name"
		if id == "b1" {
			field = "p1_medical.type"
		}
		if err := db.CreateAttachment(Attachment{
			ID: id, FieldID: field, Category: "identity",
			Filename: id, ContentType: "text/plain", Size: 1, Data: "x", CreatedAt: time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := db.DeleteAttachmentsByField("p1_identity.name"); err != nil {
		t.Fatal(err)
	}
	metas, _ := db.ListAttachmentsByField("p1_identity.name")
	if len(metas) != 0 {
		t.Fatalf("expected 0 attachments for deleted field, got %d", len(metas))
	}
	remain, _ := db.ListAttachmentsByField("p1_medical.type")
	if len(remain) != 1 {
		t.Fatalf("expected 1 remaining attachment, got %d", len(remain))
	}
}