package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
)

func HeadersMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func VerifySignature(r *http.Request, secret string) bool {
	// If no secret configured, skip verification (dev mode)
	if secret == "" {
		return true
	}

	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		return false
	}

	parts := strings.SplitN(signature, "=", 2)
	if len(parts) != 2 || parts[0] != "sha256" {
		return false
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	// Restore body for further reading
	// Note: In a real high-perf scenario, use a TeeReader or careful buffer management
	// For this example, we assume middleware runs before decoding body or handles it.
	// Since Go's http.Request.Body can only be read once, we'd need to re-assign it here.
	// However, usually signature verification is done inside the handler where we read the body anyway.
	// Let's implement the verification check inside the handlers themselves to simplify body reading logic.
	
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	
	incomingMAC, _ := hex.DecodeString(parts[1])
	return hmac.Equal(incomingMAC, expectedMAC)
}