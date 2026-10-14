package middleware

import (
	"bytes"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/k057ya/go-metrics/internal/service"
)

type SignedResponseWriter struct {
	http.ResponseWriter
	bodyBuf   bytes.Buffer
	Signature string
}

func (srw *SignedResponseWriter) Write(data []byte) (int, error) {
	n, err := srw.ResponseWriter.Write(data)
	if err != nil {
		return 0, err
	}
	_, err = srw.bodyBuf.Write(data)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func CheckAndSign(signatureKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		signFn := func(w http.ResponseWriter, r *http.Request) {

			if signatureKey != "" && r.Header.Get("HashSHA256") != "" {
				//
				// проверить подпись запроса
				//
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "cannot read request body", http.StatusBadRequest)
					return
				}
				// Восстанавливаем поток, чтобы обработчик тоже мог прочитать тело.
				r.Body = io.NopCloser(bytes.NewReader(body))

				hash, err := hex.DecodeString(r.Header.Get("HashSHA256"))
				if err != nil || !service.CheckSignature(body, signatureKey, hash) {
					http.Error(w, "invalid signature", http.StatusBadRequest)
					return
				}

				//
				// отдать ответ
				//
				srw := &SignedResponseWriter{ResponseWriter: w}

				// вызвать запрос
				next.ServeHTTP(srw, r)

				// подписать тело
				srw.Header().Add(
					"HashSHA256",
					service.GenSignature(srw.bodyBuf.Bytes(), signatureKey),
				)
			}
		}

		return http.HandlerFunc(signFn)
	}
}
