package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeUsername(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "alice.user", NormalizeUsername("  Alice.User  "))
}

func TestValidateUsername(t *testing.T) {
	t.Parallel()
	assert.True(t, ValidateUsername("alice_123"))
	assert.False(t, ValidateUsername("al"))
	assert.False(t, ValidateUsername("alice user"))
	assert.False(t, ValidateUsername("-alice"))
}

func TestLoginUsernameCandidates(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"alice"}, LoginUsernameCandidates("alice"))
	assert.Equal(t, []string{"alice", "Alice"}, LoginUsernameCandidates(" Alice "))
}
