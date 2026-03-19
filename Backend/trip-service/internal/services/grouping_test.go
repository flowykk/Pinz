package services

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentTypeToExt(t *testing.T) {
	cases := map[string]string{
		"image/jpeg":      ".jpg",
		"image/jpg":       ".jpg",
		"image/png":       ".png",
		"image/heic":      ".heic",
		"video/mp4":       ".mp4",
		"video/quicktime": ".mov",
		"other":           ".bin",
		"empty":           ".bin",
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			input := name
			if name == "empty" {
				input = ""
			}
			got := contentTypeToExt(input)
			require.Equal(t, want, got, "contentTypeToExt(%q)", input)
		})
	}
}
