package worker

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsBusyGroupErr(t *testing.T) {
	cases := map[string]struct {
		err error
		want bool
	}{
		"nil": {nil, false},
		"plain_error": {errors.New("something failed"), false},
		"busygroup_in_message": {errors.New("BUSYGROUP Consumer Group name already exists"), true},
		"busygroup_prefix": {errors.New("BUSYGROUP x"), true},
		"busygroup_lowercase": {errors.New("busygroup x"), false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := isBusyGroupErr(tc.err)
			require.Equal(t, tc.want, got)
		})
	}
}
