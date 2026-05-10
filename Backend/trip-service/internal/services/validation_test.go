package services

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCategory(t *testing.T) {
	cases := map[string]struct {
		input string
		want bool
	}{
		"vacation": {"vacation", true},
		"business": {"business", true},
		"holidays": {"holidays", true},
		"Активный_отдых": {"active", true},
		"education": {"education", true},
		"custom": {"custom", true},
		"invalid": {"invalid", false},
		"empty": {"", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := validateCategory(tc.input)
			require.Equal(t, tc.want, got, "validateCategory(%q)", tc.input)
		})
	}
}

func TestValidateSeason(t *testing.T) {
	cases := map[string]struct {
		input string
		want bool
	}{
		"winter": {"winter", true},
		"spring": {"spring", true},
		"summer": {"summer", true},
		"autumn": {"autumn", true},
		"invalid": {"invalid", false},
		"empty": {"", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := validateSeason(tc.input)
			require.Equal(t, tc.want, got, "validateSeason(%q)", tc.input)
		})
	}
}

func TestValidatePrivacyLevel(t *testing.T) {
	cases := map[string]struct {
		input string
		want bool
	}{
		"public": {"public", true},
		"private": {"private", true},
		"restricted": {"restricted", true},
		"invalid": {"invalid", false},
		"empty": {"", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := validatePrivacyLevel(tc.input)
			require.Equal(t, tc.want, got, "validatePrivacyLevel(%q)", tc.input)
		})
	}
}

func TestValidateContentType(t *testing.T) {
	cases := map[string]struct {
		input string
		want bool
	}{
		"image/jpeg": {"image/jpeg", true},
		"image/jpg": {"image/jpg", true},
		"image/png": {"image/png", true},
		"image/heic": {"image/heic", true},
		"video/mp4": {"video/mp4", true},
		"video/quicktime": {"video/quicktime", true},
		"image/gif": {"image/gif", false},
		"video/avi": {"video/avi", false},
		"application/pdf": {"application/pdf", false},
		"empty": {"", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := validateContentType(tc.input)
			require.Equal(t, tc.want, got, "validateContentType(%q)", tc.input)
		})
	}
}

func TestValidateTags(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		err := validateTags([]string{"beach", "sunset", "travel"})
		require.NoError(t, err)
	})
	t.Run("too_many_tags", func(t *testing.T) {
		tags := make([]string, 11)
		for i := range tags {
			tags[i] = "tag"
		}
		err := validateTags(tags)
		require.Error(t, err)
	})
	t.Run("tag_too_long", func(t *testing.T) {
		err := validateTags([]string{"this-is-a-very-long-tag-name"})
		require.Error(t, err)
	})
	t.Run("empty_list_ok", func(t *testing.T) {
		err := validateTags([]string{})
		require.NoError(t, err)
	})
	t.Run("exactly_max", func(t *testing.T) {
		tags := make([]string, 10)
		for i := range tags {
			tags[i] = "tag"
		}
		err := validateTags(tags)
		require.NoError(t, err)
	})
	t.Run("tag_exactly_15_chars", func(t *testing.T) {
		err := validateTags([]string{"123456789012345"})
		require.NoError(t, err)
	})
}

func TestValidatePinCategory(t *testing.T) {
	cases := map[string]struct {
		input string
		want string
	}{
		"valid": {"sight", "sight"},
		"food": {"food", "food"},
		"other": {"custom", "custom"},
		"unknown": {"Unknown", "custom"},
		"empty": {"", "custom"},
		"english": {"Sightseeing", "custom"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := ValidatePinCategory(tc.input)
			require.Equal(t, tc.want, got)
		})
	}
}
