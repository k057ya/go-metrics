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
	bodyBuf    bytes.Buffer
	statusCode int
}

func (srw *SignedResponseWriter) Write(data []byte) (int, error) {
	if srw.statusCode == 0 {
		srw.WriteHeader(http.StatusOK)
	}
	return srw.bodyBuf.Write(data)
}

func (srw *SignedResponseWriter) WriteHeader(statusCode int) {
	if srw.statusCode == 0 {
		srw.statusCode = statusCode
	}
}

func CheckAndSign(signatureKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		signFn := func(w http.ResponseWriter, r *http.Request) {

			if signatureKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			if signature := r.Header.Get("HashSHA256"); signature != "" {
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

				hash, err := hex.DecodeString(signature)
				if err != nil || !service.CheckSignature(body, signatureKey, hash) {
					http.Error(w, "invalid signature", http.StatusBadRequest)
					return
				}
			}

			// Вызываем обработчик независимо от наличия подписи в запросе.
			// Буферизуем ответ, чтобы подписать его до отправки заголовков.
			srw := &SignedResponseWriter{ResponseWriter: w}
			next.ServeHTTP(srw, r)
			w.Header().Set("HashSHA256", service.GenSignature(srw.bodyBuf.Bytes(), signatureKey))
			if srw.statusCode == 0 {
				srw.statusCode = http.StatusOK
			}
			w.WriteHeader(srw.statusCode)
			w.Write(srw.bodyBuf.Bytes())
		}

		return http.HandlerFunc(signFn)
	}
}
