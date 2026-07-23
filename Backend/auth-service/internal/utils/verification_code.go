package utils

import (
	"crypto/rand"
	"fmt"
)

func GenerateVerificationCode() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	if n < 0 {
		n = -n
	}
	return fmt.Sprintf("%04d", n%10000)
}
