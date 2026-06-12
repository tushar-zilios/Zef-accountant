package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return []byte("zef-super-secret-jwt-key-change-in-production")
	}
	return []byte(secret)
}

func getJWTExpirationHours() int {
	expStr := os.Getenv("JWT_EXPIRATION_HOURS")
	if expStr == "" {
		return 24
	}
	hours, err := strconv.Atoi(expStr)
	if err != nil {
		return 24
	}
	return hours
}

type Claims struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	Exp     int64  `json:"exp"`
	PwdHash string `json:"pwd_hash,omitempty"`
}

func base64URLEncode(b []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
}

func base64URLDecode(s string) ([]byte, error) {
	// Re-add padding if needed
	if l := len(s) % 4; l > 0 {
		s += strings.Repeat("=", 4-l)
	}
	return base64.URLEncoding.DecodeString(s)
}

// GenerateToken generates a signed HS256 JWT containing user claims
func GenerateToken(userID string, email string, passwordHash *string) (string, error) {
	// Header
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	headerEnc := base64URLEncode(headerBytes)

	var pwdHash string
	if passwordHash != nil {
		pwdHash = *passwordHash
	}

	// Payload (Expires in configured hours)
	claims := Claims{
		UserID:  userID,
		Email:   email,
		Exp:     time.Now().Add(time.Duration(getJWTExpirationHours()) * time.Hour).Unix(),
		PwdHash: pwdHash,
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	claimsEnc := base64URLEncode(claimsBytes)

	// Signature
	unsignedToken := headerEnc + "." + claimsEnc
	h := hmac.New(sha256.New, getJWTSecret())
	h.Write([]byte(unsignedToken))
	signature := base64URLEncode(h.Sum(nil))

	return unsignedToken + "." + signature, nil
}


// VerifyToken validates the signature and expiration of the JWT
func VerifyToken(tokenString string) (*Claims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	unsignedToken := parts[0] + "." + parts[1]
	signature, err := base64URLDecode(parts[2])
	if err != nil {
		return nil, err
	}

	// Verify signature matches secret
	h := hmac.New(sha256.New, getJWTSecret())
	h.Write([]byte(unsignedToken))
	expectedSignature := h.Sum(nil)

	if !hmac.Equal(signature, expectedSignature) {
		return nil, errors.New("invalid signature")
	}

	// Decode claims payload
	payloadBytes, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, err
	}

	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, err
	}

	// Verify token expiration
	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("token expired")
	}

	return &claims, nil
}
