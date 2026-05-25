package auth

import (
	"regexp"
	"strings"
)

const UsernameValidationMessage = "Use 3-32 characters: lowercase letters, numbers, dots, underscores, or hyphens. Start with a letter or number."

var usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,31}$`)

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
