package utils

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

var digitsOnly = regexp.MustCompile(`^\d{4}$`)

func TestGenerateVerificationCode(t *testing.T) {
	cases := map[string]func(t *testing.T){
		"length_4": func(t *testing.T) {
			code := GenerateVerificationCode()
			require.Len(t, code, 4)
		},
		"only_digits": func(t *testing.T) {
			code := GenerateVerificationCode()
			require.True(t, digitsOnly.MatchString(code), "code %q must be 4 digits", code)
		},
		"in_range_0_9999": func(t *testing.T) {
			code := GenerateVerificationCode()
			n, err := strconv.Atoi(code)
			require.NoError(t, err)
			require.GreaterOrEqual(t, n, 0)
			require.LessOrEqual(t, n, 9999)
		},
	}
	for name, fn := range cases {
		t.Run(name, fn)
	}
}
