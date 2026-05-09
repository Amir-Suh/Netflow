package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Claims struct {
	Subject   string `json:"sub"`
	Email     string `json:"email"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func NewJWT(secret []byte, userID int64, email string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		Subject:   strconv.FormatInt(userID, 10),
		Email:     email,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
	}
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	signature := sign(secret, unsigned)
	return unsigned + "." + signature, nil
}

func ParseJWT(secret []byte, token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("invalid token format")
	}
	headerPayload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, fmt.Errorf("decode token header: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerPayload, &header); err != nil {
		return Claims{}, fmt.Errorf("parse token header: %w", err)
	}
	if header.Alg != "HS256" || header.Typ != "JWT" {
		return Claims{}, errors.New("unsupported token header")
	}
	unsigned := parts[0] + "." + parts[1]
	expected := sign(secret, unsigned)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return Claims{}, errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("decode token payload: %w", err)
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, fmt.Errorf("parse token payload: %w", err)
	}
	if claims.ExpiresAt <= time.Now().UTC().Unix() {
		return Claims{}, errors.New("token expired")
	}
	if claims.Subject == "" {
		return Claims{}, errors.New("token subject is required")
	}
	return claims, nil
}

func sign(secret []byte, unsigned string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
