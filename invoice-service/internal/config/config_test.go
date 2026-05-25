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
	if cfg.RegistrationEnabled {
		t.Error("RegistrationEnabled should be false by default")
	}
}

func TestConfigEnvOverrides(t *testing.T) {
	cleanEnv(t)
	mustSetenv(t, "ADDRESS", ":9090")
	mustSetenv(t, "LOG_LEVEL", "debug")
	mustSetenv(t, "LOG_FORMAT", "text")
	mustSetenv(t, "LOG_INCLUDE_SOURCE", "true")
	mustSetenv(t, "DATABASE_PATH", "/tmp/test.db")
	mustSetenv(t, "MIGRATIONS_DIR", "/tmp/migrations")

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

	mustSetenv(t, "OCR_ENABLED", "true")
	cfg := Load()
	if !cfg.OCREnabled {
		t.Error("OCREnabled should be true when OCR_ENABLED=true")
	}

	mustSetenv(t, "OCR_ENABLED", "false")
	cfg = Load()
	if cfg.OCREnabled {
		t.Error("OCREnabled should be false when OCR_ENABLED=false")
	}

	mustUnsetenv(t, "OCR_ENABLED")
	cfg = Load()
	if cfg.OCREnabled {
		t.Error("OCREnabled should be false when OCR_ENABLED is empty")
	}
}

func TestRegistrationEnabled(t *testing.T) {
	cleanEnv(t)

	mustSetenv(t, "REGISTRATION_ENABLED", "true")
	cfg := Load()
	if !cfg.RegistrationEnabled {
		t.Error("RegistrationEnabled should be true when REGISTRATION_ENABLED=true")
	}

	mustSetenv(t, "REGISTRATION_ENABLED", "false")
	cfg = Load()
	if cfg.RegistrationEnabled {
		t.Error("RegistrationEnabled should be false when REGISTRATION_ENABLED=false")
	}

	mustUnsetenv(t, "REGISTRATION_ENABLED")
	cfg = Load()
	if cfg.RegistrationEnabled {
		t.Error("RegistrationEnabled should be false when REGISTRATION_ENABLED is empty")
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
		"AUTH_ENABLED", "REGISTRATION_ENABLED", "AUTH_DB_PATH", "AUTH_MIGRATIONS_DIR",
		"USER_DB_DIR", "WEB_AUTHN_RP_ID", "WEB_AUTHN_RP_ORIGINS",
		"WEB_AUTHN_RP_DISPLAY", "SESSION_TTL", "TRUSTED_PROXY",
	} {
		mustUnsetenv(t, key)
	}
}

func mustSetenv(t *testing.T, key, value string) {
	t.Helper()
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Setenv %s: %v", key, err)
	}
}

func mustUnsetenv(t *testing.T, key string) {
	t.Helper()
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv %s: %v", key, err)
	}
}
