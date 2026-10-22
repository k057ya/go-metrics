package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func GenSignature(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func CheckSignature(data []byte, key string, hash []byte) bool {
	expected, err := hex.DecodeString(GenSignature(data, key))
	return err == nil && hmac.Equal(hash, expected)
}
