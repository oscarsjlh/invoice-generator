package config

import (
	"os"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	t.Parallel()
	// Clear any env vars that might affect defaults
	cleanEnv(t)

	cfg := Load()

	if cfg.Address != ":8080" {
		t.Errorf("Address = %q, want %q", cfg.Address, ":8080")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "json")
	}
	if cfg.LogIncludeSource {
		t.Error("LogIncludeSource should be false by default")
	}
	if cfg.SMTPPort != "587" {
		t.Errorf("SMTPPort = %q, want %q", cfg.SMTPPort, "587")
	}
	if cfg.OCREnabled {
		t.Error("OCREnabled should be false by default")
	}
}

func TestConfigEnvOverrides(t *testing.T) {
	cleanEnv(t)
	os.Setenv("ADDRESS", ":9090")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FORMAT", "text")
	os.Setenv("LOG_INCLUDE_SOURCE", "true")
	os.Setenv("DATABASE_PATH", "/tmp/test.db")
	os.Setenv("MIGRATIONS_DIR", "/tmp/migrations")

	cfg := Load()

	if cfg.Address != ":9090" {
		t.Errorf("Address = %q, want %q", cfg.Address, ":9090")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "text")
	}
	if !cfg.LogIncludeSource {
		t.Error("LogIncludeSource should be true when LOG_INCLUDE_SOURCE=true")
	}
	if cfg.DatabasePath != "/tmp/test.db" {
		t.Errorf("DatabasePath = %q, want %q", cfg.DatabasePath, "/tmp/test.db")
	}
	if cfg.MigrationsDir != "/tmp/migrations" {
		t.Errorf("MigrationsDir = %q, want %q", cfg.MigrationsDir, "/tmp/migrations")
	}
}

func TestOCREnabled(t *testing.T) {
	cleanEnv(t)

	os.Setenv("OCR_ENABLED", "true")
	cfg := Load()
	if !cfg.OCREnabled {
		t.Error("OCREnabled should be true when OCR_ENABLED=true")
	}

	os.Setenv("OCR_ENABLED", "false")
	cfg = Load()
	if cfg.OCREnabled {
		t.Error("OCREnabled should be false when OCR_ENABLED=false")
	}

	os.Unsetenv("OCR_ENABLED")
	cfg = Load()
	if cfg.OCREnabled {
		t.Error("OCREnabled should be false when OCR_ENABLED is empty")
	}
}

func TestSMTPConfig(t *testing.T) {
	t.Parallel()
	cleanEnv(t)

	cfg := Load()
	if cfg.SMTPHost != "" {
		t.Errorf("SMTPHost should be empty by default, got %q", cfg.SMTPHost)
	}
	if cfg.SMTPUser != "" {
		t.Errorf("SMTPUser should be empty by default, got %q", cfg.SMTPUser)
	}
	if cfg.SMTPPass != "" {
		t.Errorf("SMTPPass should be empty by default, got %q", cfg.SMTPPass)
	}
	if cfg.SMTPFrom != "" {
		t.Errorf("SMTPFrom should be empty by default, got %q", cfg.SMTPFrom)
	}
}

func cleanEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"ADDRESS", "DATABASE_PATH", "MIGRATIONS_DIR", "TYPST_BIN",
		"SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASS", "SMTP_FROM",
		"OCR_SERVICE_URL", "OCR_ENABLED", "OCR_UPLOAD_DIR",
		"LOG_LEVEL", "LOG_FORMAT", "LOG_INCLUDE_SOURCE",
	} {
		os.Unsetenv(key)
	}
}
