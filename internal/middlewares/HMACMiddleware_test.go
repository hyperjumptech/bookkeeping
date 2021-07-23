package middlewares

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"
)

func TestHMACMiddleware(t *testing.T) {
	hmac := GenHMAC()
	// t.Logf("%s", hmac)
	if ValidateHMAC(hmac) != true {
		t.FailNow()
	}
}

func GenOldHMAC() string {
	time := time.Now().Add(time.Duration(-1*(HMACAgeMinutes+1)) * time.Minute).Format(time.RFC3339)
	hash := ComputeHmac(time, SecretKey)
	toBase := fmt.Sprintf("%s$%s", time, hash)
	base64hmac := base64.StdEncoding.EncodeToString([]byte(toBase))
	return base64hmac
}

func TestOldHMAC(t *testing.T) {
	if ValidateHMAC(GenOldHMAC()) != false {
		t.FailNow()
	}
}
