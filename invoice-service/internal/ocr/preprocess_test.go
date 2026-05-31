package ocr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateImageFileAcceptsHEICSignature(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "photo.heic")
	data := []byte{
		0x00, 0x00, 0x00, 0x18,
		'f', 't', 'y', 'p',
		'h', 'e', 'i', 'c',
		0x00, 0x00, 0x00, 0x00,
		'h', 'e', 'i', 'c',
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write heic fixture: %v", err)
	}

	if err := ValidateImageFile(path); err != nil {
		t.Fatalf("ValidateImageFile returned error: %v", err)
	}
}

func TestValidateImageFileRejectsInvalidHEIC(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "photo.heic")
	if err := os.WriteFile(path, []byte("not a heic file"), 0o644); err != nil {
		t.Fatalf("write invalid heic fixture: %v", err)
	}

	if err := ValidateImageFile(path); err == nil {
		t.Fatal("ValidateImageFile returned nil error for invalid HEIC")
	}
}
