// Package gosig implementa la firma HMAC entre la API y el processor. La usan los dos, así que
// una firma que cambie aquí cambia en ambos lados a la vez.
package gosig

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strconv"
	"time"
)

const (
	HeaderSignature = "x-soundboard-signature"
	HeaderTimestamp = "x-soundboard-timestamp"
	MaxAgeSeconds   = 300
)

func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Firma `${timestamp}.${sha256(body)}` con HMAC-SHA256, en hexadecimal.
func sign(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + SHA256Hex(body)))
	return hex.EncodeToString(mac.Sum(nil))
}

func Headers(secret string, body []byte, now time.Time) map[string]string {
	timestamp := strconv.FormatInt(now.Unix(), 10)
	return map[string]string{HeaderTimestamp: timestamp, HeaderSignature: sign(secret, timestamp, body)}
}

func Verify(secret string, body []byte, timestamp, signature string, now time.Time) bool {
	if timestamp == "" || signature == "" {
		return false
	}
	seconds, err := strconv.ParseFloat(timestamp, 64)
	if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return false
	}
	if math.Abs(float64(now.UnixMilli())/1000-seconds) > MaxAgeSeconds {
		return false
	}
	return hmac.Equal([]byte(sign(secret, timestamp, body)), []byte(signature))
}
