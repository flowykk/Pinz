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

func TestParseLocation(t *testing.T) {
	loc := parseLocation(map[string]any{
		"id": float64(10),
		"parent_id": float64(5),
		"name": "Paris",
		"type": "City",
	})
	require.NotNil(t, loc)
	require.EqualValues(t, 10, loc.ID)
	require.NotNil(t, loc.ParentID)
	require.EqualValues(t, 5, *loc.ParentID)
	require.Equal(t, "Paris", loc.Name)
	require.Equal(t, "City", loc.Type)
}

func TestParseLocation_NoParent(t *testing.T) {
	loc := parseLocation(map[string]any{"id": float64(1), "name": "X", "type": "Country"})
	require.NotNil(t, loc)
	require.Nil(t, loc.ParentID)
}
