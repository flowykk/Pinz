package services

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateUsernameFormat(t *testing.T) {
	cases := map[string]struct {
		input string
		want  bool
	}{
		"letters":          {"JohnDoe", true},
		"digits":           {"user123", true},
		"underscore":       {"my_user", true},
		"hyphen":           {"my-user", true},
		"mixed":            {"User_name-01", true},
		"spaces":           {"user name", false},
		"cyrillic":         {"Пользователь", false},
		"special_chars":    {"user@name", false},
		"dot":              {"user.name", false},
		"exclamation":      {"user!", false},
		"empty":            {"", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := validateUsernameFormat(tc.input)
			require.Equal(t, tc.want, got, "validateUsernameFormat(%q)", tc.input)
		})
	}
}
