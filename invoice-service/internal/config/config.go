package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Address          string
	DatabasePath     string
	MigrationsDir    string
	TypstBin         string
	SMTPHost         string
	SMTPPort         string
	SMTPUser         string
	SMTPPass         string
	SMTPFrom         string
	OCRServiceURL    string
	OCREnabled       bool
	OCRUploadDir     string
	DefaultDueDays   int
	LogLevel         string
	LogFormat        string
	LogIncludeSource bool

	// Auth configuration
	AuthEnabled         bool
	RegistrationEnabled bool
	AuthDBPath          string
	AuthMigrationsDir   string
	UserDBDir           string
	WebAuthnRPID        string
	WebAuthnRPOrigins   []string
	WebAuthnRPDisplay   string
	SessionTTL          time.Duration
	TrustedProxy        bool
}

func Load() Config {
	return Config{
		Address:          getenv("ADDRESS", ":8080"),
		DatabasePath:     getenv("DATABASE_PATH", "data/invoices.db"),
		MigrationsDir:    getenv("MIGRATIONS_DIR", "migrations"),
		TypstBin:         getenv("TYPST_BIN", ""),
		SMTPHost:         getenv("SMTP_HOST", ""),
		SMTPPort:         getenv("SMTP_PORT", "587"),
		SMTPUser:         getenv("SMTP_USER", ""),
		SMTPPass:         getenv("SMTP_PASS", ""),
		SMTPFrom:         getenv("SMTP_FROM", ""),
		OCRServiceURL:    getenv("OCR_SERVICE_URL", ""),
		OCREnabled:       getenv("OCR_ENABLED", "") == "true",
		OCRUploadDir:     getenv("OCR_UPLOAD_DIR", "data/ocr-uploads"),
		LogLevel:         getenv("LOG_LEVEL", "info"),
		LogFormat:        getenv("LOG_FORMAT", "json"),
		LogIncludeSource: getenv("LOG_INCLUDE_SOURCE", "") == "true",

		AuthEnabled:         getenv("AUTH_ENABLED", "") != "false",
		RegistrationEnabled: getenv("REGISTRATION_ENABLED", "") == "true",
		AuthDBPath:          getenv("AUTH_DB_PATH", "data/auth.db"),
		AuthMigrationsDir:   getenv("AUTH_MIGRATIONS_DIR", "auth-migrations"),
		UserDBDir:           getenv("USER_DB_DIR", "data/users"),
		WebAuthnRPID:        getenv("WEB_AUTHN_RP_ID", "localhost"),
		WebAuthnRPOrigins:   getenvSlice("WEB_AUTHN_RP_ORIGINS", []string{"http://localhost:8080"}),
		WebAuthnRPDisplay:   getenv("WEB_AUTHN_RP_DISPLAY", "Invoice App"),
		SessionTTL:          getenvDuration("SESSION_TTL", 24*time.Hour),
		TrustedProxy:        getenv("TRUSTED_PROXY", "") == "true",
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvSlice(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return []string{value}
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	d, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return time.Duration(d) * time.Second
}
