package room

import (
	"math/rand"
)

const codeLength = 4
const maxRetries = 100

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

// GenerateCode creates a random 4-letter uppercase room code.
// It checks against existing codes to avoid duplicates.
func GenerateCode(existing map[string]bool) string {
	for range maxRetries {
		code := randomCode()
		if !existing[code] {
			return code
		}
	}
	// Fallback: extremely unlikely with 26^4 = 456,976 combinations
	return randomCode()
}

func randomCode() string {
	b := make([]rune, codeLength)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
