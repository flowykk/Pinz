package worker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseUserIDs(t *testing.T) {
	require.Nil(t, parseUserIDs(nil))
	require.Nil(t, parseUserIDs(""))
	require.Nil(t, parseUserIDs("not json"))
	require.Equal(t, []string{"a", "b"}, parseUserIDs(`["a","b"]`))
}

func TestParsePayload(t *testing.T) {
	require.Equal(t, map[string]any{}, parsePayload(nil))
	require.Equal(t, map[string]any{"a": float64(1)}, parsePayload(`{"a":1}`))
}

func TestReadFloat(t *testing.T) {
	cases := []struct {
		in   any
		want float64
		ok   bool
	}{
		{float64(1.5), 1.5, true},
		{float32(2.5), 2.5, true},
		{int(3), 3, true},
		{int32(4), 4, true},
		{int64(5), 5, true},
		{"x", 0, false},
		{nil, 0, false},
	}
	for _, c := range cases {
		got, ok := readFloat(c.in)
		require.Equal(t, c.ok, ok)
		require.InDelta(t, c.want, got, 0.0001)
	}
}
