package crypto

import (
	"crypto/rand"
	"fmt"
)

// Character set with uppercase, lowercase and digits.
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GeneratePassword generates a random password of the specified length using the defined character set.
func GeneratePassword(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be positive")
	}

	password := make([]byte, length)

	for i := range password {
		randomByte := make([]byte, 1)
		if _, err := rand.Read(randomByte); err != nil {
			return "", err
		}
		password[i] = charset[int(randomByte[0])%len(charset)]
	}

	return string(password), nil
}
