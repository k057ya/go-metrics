package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

func GenSignature(data []byte, key string) []byte {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return h.Sum(nil)
}

func GenSignatureBase64(data []byte, key string) string {
	return base64.StdEncoding.EncodeToString(GenSignature(data, key))
}

func CheckSignature(data []byte, key string, hash []byte) bool {
	return hmac.Equal(hash, GenSignature(data, key))
}

func CheckSignatureBase64(data []byte, key string, signature string) bool {
	hash, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}
	return CheckSignature(data, key, hash)
}
