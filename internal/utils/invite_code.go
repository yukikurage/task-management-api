package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateInviteCode generates a random invite code in the format XXXX-XXXX-XXXX
func GenerateInviteCode() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	hex := hex.EncodeToString(bytes)
	// Format: XXXX-XXXX-XXXX
	return fmt.Sprintf("%s-%s-%s",
		hex[0:4],
		hex[4:8],
		hex[8:12],
	), nil
}
