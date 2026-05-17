package config

import "os"

type Config struct {
	Address       string
	DatabasePath  string
	MigrationsDir string
	TypstBin      string
	SMTPHost      string
	SMTPPort      string
	SMTPUser      string
	SMTPPass      string
	SMTPFrom      string
	OCRServiceURL string
	OCREnabled    bool
	OCRUploadDir  string
	LogLevel      string
	LogFormat     string
}

func Load() Config {
	return Config{
		Address:       getenv("ADDRESS", ":8080"),
		DatabasePath:  getenv("DATABASE_PATH", "data/invoices.db"),
		MigrationsDir: getenv("MIGRATIONS_DIR", "migrations"),
		TypstBin:      getenv("TYPST_BIN", ""),
		SMTPHost:      getenv("SMTP_HOST", ""),
		SMTPPort:      getenv("SMTP_PORT", "587"),
		SMTPUser:      getenv("SMTP_USER", ""),
		SMTPPass:      getenv("SMTP_PASS", ""),
		SMTPFrom:      getenv("SMTP_FROM", ""),
		OCRServiceURL: getenv("OCR_SERVICE_URL", ""),
		OCREnabled:    getenv("OCR_ENABLED", "") == "true",
		OCRUploadDir:  getenv("OCR_UPLOAD_DIR", "data/ocr-uploads"),
		LogLevel:      getenv("LOG_LEVEL", "info"),
		LogFormat:     getenv("LOG_FORMAT", "json"),
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
