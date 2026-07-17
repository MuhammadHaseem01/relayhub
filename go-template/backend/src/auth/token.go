package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type Payload struct {
	ID      int
	Subject int
	Issued  int64
	Expiry  int64
}

func CreateToken(userID int, secret string) string {
	header := encode(map[string]any{"alg": "HS256", "typ": "JWT"})
	payload := encode(map[string]any{"id": userID, "iat": time.Now().Unix()})
	unsigned := header + "." + payload
	return unsigned + "." + sign(unsigned, secret)
}

func VerifyToken(token string, secret string) (Payload, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 && len(parts) != 3 {
		return Payload{}, false
	}
	encodedPayload := parts[0]
	if len(parts) == 3 {
		encodedPayload = parts[1]
	}
	signature := parts[len(parts)-1]
	signedValue := strings.Join(parts[:len(parts)-1], ".")
	if !hmac.Equal([]byte(signature), []byte(sign(signedValue, secret))) {
		return Payload{}, false
	}
	return decodePayload(encodedPayload)
}

func sign(value string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encode(value any) string {
	raw, _ := json.Marshal(value)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodePayload(encoded string) (Payload, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Payload{}, false
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return Payload{}, false
	}
	userID, ok := numericID(body["id"])
	if !ok {
		userID, ok = numericID(body["sub"])
	}
	if !ok || userID <= 0 {
		return Payload{}, false
	}
	exp, _ := int64Value(body["exp"])
	if exp > 0 && exp < time.Now().Unix() {
		return Payload{}, false
	}
	issued, _ := int64Value(body["iat"])
	return Payload{ID: userID, Subject: userID, Issued: issued, Expiry: exp}, true
}

func numericID(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		if v == float64(int(v)) {
			return int(v), true
		}
	case string:
		id, err := strconv.Atoi(v)
		return id, err == nil
	}
	return 0, false
}

func int64Value(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		return parsed, err == nil
	}
	return 0, false
}
