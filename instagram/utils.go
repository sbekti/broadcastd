package instagram

import (
	"crypto/hmac"
	"crypto/md5"
	cryptorand "crypto/rand"
	"crypto/sha256"
	b64 "encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"time"
	"unsafe"
)

const (
	volatileSeed = "12345"
)

func byteToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func getRandom(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

func generateMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func generateHMAC(text, key string) string {
	hasher := hmac.New(sha256.New, []byte(key))
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func generateDeviceID(seed string) string {
	hash := generateMD5Hash(seed + volatileSeed)
	return "android-" + hash[:16]
}

func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(cryptorand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

func generateUUID() string {
	uuid, err := newUUID()
	if err != nil {
		return "cb479ee7-a50d-49e7-8b7b-60cc1a105e22" // default value when error occurred
	}
	return uuid
}

func generateSignature(data string) map[string]string {
	m := make(map[string]string)
	m["ig_sig_key_version"] = igSigKeyVersion
	m["signed_body"] = fmt.Sprintf(
		"%s.%s", generateHMAC(data, igSigKey), data,
	)
	return m
}

func randInt(min int, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min
}

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func generateBreadcrumb(size int) string {
	timeElapsed := randInt(500, 1500) + size + randInt(500, 1500)
	textChangeEventCount := max(1, size/randInt(3, 5))
	dt := time.Now().Unix() * 1000

	data := fmt.Sprintf("%d %d %d %d", size, timeElapsed, textChangeEventCount, dt)
	body := b64.StdEncoding.EncodeToString([]byte(data))

	h := hmac.New(sha256.New, []byte(breadcrumbKey))
	h.Write([]byte(data))
	sig := b64.StdEncoding.EncodeToString(h.Sum(nil))

	return sig + "\n" + body
}
