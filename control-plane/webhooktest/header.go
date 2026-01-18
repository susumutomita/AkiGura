package webhooktest

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"time"
)

// NewSignedHeader generates a Stripe-style "t=...,v1=..." header for tests.
func NewSignedHeader(body []byte, secret string) string {
	ts := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.", ts)
	mac.Write(body)
	sig := mac.Sum(nil)
	return fmt.Sprintf("t=%d,v1=%x", ts, sig)
}
