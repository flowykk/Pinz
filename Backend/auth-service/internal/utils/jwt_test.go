package utils

import (
	"encoding/base64"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestGenerateAccessToken(t *testing.T) {
	cases := map[string]func(t *testing.T){
		"valid_returns_non_empty_token": func(t *testing.T) {
			tok, err := GenerateAccessToken("user-1", "alice", "secret")
			require.NoError(t, err)
			require.NotEmpty(t, tok)
		},
		"token_contains_user_id_and_username_claims": func(t *testing.T) {
			tok, err := GenerateAccessToken("user-42", "bob", "key")
			require.NoError(t, err)
			parsed, _, err := jwt.NewParser().ParseUnverified(tok, jwt.MapClaims{})
			require.NoError(t, err)
			claims, ok := parsed.Claims.(jwt.MapClaims)
			require.True(t, ok)
			require.Equal(t, "user-42", claims["user_id"])
			require.Equal(t, "bob", claims["username"])
			require.NotZero(t, claims["exp"])
			require.NotZero(t, claims["iat"])
		},
	}
	for name, fn := range cases {
		t.Run(name, fn)
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	cases := map[string]func(t *testing.T){
		"non_empty": func(t *testing.T) {
			tok, err := GenerateRefreshToken()
			require.NoError(t, err)
			require.NotEmpty(t, tok)
		},
		"valid_base64": func(t *testing.T) {
			tok, err := GenerateRefreshToken()
			require.NoError(t, err)
			_, err = base64.StdEncoding.DecodeString(tok)
			require.NoError(t, err)
		},
		"decodes_to_32_bytes": func(t *testing.T) {
			tok, err := GenerateRefreshToken()
			require.NoError(t, err)
			raw, err := base64.StdEncoding.DecodeString(tok)
			require.NoError(t, err)
			require.Len(t, raw, 32)
		},
	}
	for name, fn := range cases {
		t.Run(name, fn)
	}
}
