package s3

import (
	"os"
	"strings"
	"sync"
)

var (
	keyPrefixOnce sync.Once
	keyPrefix string
)

// KeyPrefix returns S3_KEY_PREFIX normalized to "" or "<prefix>/".
func KeyPrefix() string {
	keyPrefixOnce.Do(func() {
		p := strings.TrimSpace(os.Getenv("S3_KEY_PREFIX"))
		p = strings.Trim(p, "/")
		if p == "" {
			keyPrefix = ""
			return
		}
		keyPrefix = p + "/"
	})
	return keyPrefix
}

func PrefixedKey(key string) string {
	return KeyPrefix() + key
}
