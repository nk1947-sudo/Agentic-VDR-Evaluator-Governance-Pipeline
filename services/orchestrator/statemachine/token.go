package statemachine

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type TokenManager struct {
	secretKey []byte
}

func NewTokenManager(secret string) *TokenManager {
	return &TokenManager{
		secretKey: []byte(secret),
	}
}

// GenerateReviewToken produces a cryptographically signed review token:
// Format: <dealID>:<expiresUnix>:<hmacHex>
func (tm *TokenManager) GenerateReviewToken(dealID string, ttl time.Duration) string {
	expiresAt := time.Now().Add(ttl).Unix()
	payload := fmt.Sprintf("%s:%d", dealID, expiresAt)

	mac := hmac.New(sha256.New, tm.secretKey)
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s:%s", payload, signature)
}

// ValidateReviewToken verifies token integrity, deal match, and expiration
func (tm *TokenManager) ValidateReviewToken(token, expectedDealID string) error {
	parts := strings.Split(token, ":")
	if len(parts) != 3 {
		return errors.New("malformed review token format")
	}

	dealID := parts[0]
	expiresStr := parts[1]
	providedSig := parts[2]

	if dealID != expectedDealID {
		return fmt.Errorf("token deal ID mismatch: expected %s, got %s", expectedDealID, dealID)
	}

	expiresUnix, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil {
		return errors.New("invalid expiration in token")
	}

	if time.Now().Unix() > expiresUnix {
		return errors.New("review token has expired")
	}

	payload := fmt.Sprintf("%s:%d", dealID, expiresUnix)
	mac := hmac.New(sha256.New, tm.secretKey)
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(providedSig), []byte(expectedSig)) {
		return errors.New("invalid review token cryptographic signature")
	}

	return nil
}
