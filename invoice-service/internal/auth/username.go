package auth

import (
	"regexp"
	"strings"
)

const UsernameValidationMessage = "Use 3-32 characters: lowercase letters, numbers, dots, underscores, or hyphens. Start with a letter or number."

const usernamePatternStr = `^[a-z0-9][a-z0-9._-]{2,31}$`

var usernamePattern = regexp.MustCompile(usernamePatternStr)

type UsernamePolicy struct {
	Pattern       string `json:"pattern"`
	Message       string `json:"message"`
	Normalization string `json:"normalization"`
}

var DefaultUsernamePolicy = UsernamePolicy{
	Pattern:       usernamePatternStr,
	Message:       UsernameValidationMessage,
	Normalization: "trimmed and lowercased",
}

func NormalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func ValidateUsername(username string) bool {
	return usernamePattern.MatchString(username)
}

func LoginUsernameCandidates(username string) []string {
	trimmed := strings.TrimSpace(username)
	normalized := NormalizeUsername(username)
	if trimmed == "" || trimmed == normalized {
		return []string{normalized}
	}
	return []string{normalized, trimmed}
}
