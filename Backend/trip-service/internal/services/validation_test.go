package services

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCategory(t *testing.T) {
	cases := map[string]struct {
		input string
		want  bool
	}{
		"Отпуск":         {"Отпуск", true},
		"Командировка":   {"Командировка", true},
		"Выходные":       {"Выходные", true},
		"Активный_отдых": {"Активный отдых", true},
		"Образование":    {"Образование", true},
		"Другое":         {"Другое", true},
		"invalid":        {"invalid", false},
		"empty":          {"", false},
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
		want  bool
	}{
		"Зима":    {"Зима", true},
		"Весна":   {"Весна", true},
		"Лето":    {"Лето", true},
		"Осень":   {"Осень", true},
		"invalid": {"invalid", false},
		"empty":   {"", false},
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
		want  bool
	}{
		"Public":     {"Public", true},
		"Private":    {"Private", true},
		"Restricted": {"Restricted", true},
		"invalid":    {"invalid", false},
		"empty":      {"", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := validatePrivacyLevel(tc.input)
			require.Equal(t, tc.want, got, "validatePrivacyLevel(%q)", tc.input)
		})
	}
}
