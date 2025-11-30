package spotify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokensStruct(t *testing.T) {
	tokens := Tokens{
		AccessToken:  "access",
		TokenType:    "Bearer",
		Scope:        "user-read-currently-playing",
		ExpiresIn:    3600,
		RefreshToken: "refresh",
	}

	assert.Equal(t, "access", tokens.AccessToken)
	assert.Equal(t, "Bearer", tokens.TokenType)
	assert.Equal(t, "user-read-currently-playing", tokens.Scope)
	assert.Equal(t, 3600, tokens.ExpiresIn)
	assert.Equal(t, "refresh", tokens.RefreshToken)
}
