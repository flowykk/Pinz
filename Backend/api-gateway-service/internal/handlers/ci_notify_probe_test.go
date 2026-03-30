package handlers

import "testing"

// Remove this file after verifying Telegram CI failure notification works.
func TestIntentionalFailureForTelegramNotifyProbe(t *testing.T) {
	t.Fatal("intentional CI failure — delete ci_notify_probe_test.go after testing notify")
}
