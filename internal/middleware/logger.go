package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/k057ya/go-metrics/internal/logger"
	"go.uber.org/zap"
)

type ResponseData struct {
	size   int
	status int
}

type LoggableResponseWriter struct {
	http.ResponseWriter
	ResponseData *ResponseData
	wroteHeader  bool
}

func (lw LoggableResponseWriter) WriteHeader(code int) {
	if lw.wroteHeader {
		return
	}

	lw.ResponseWriter.WriteHeader(code)

	if code >= 100 && code < 200 {
		return
	}

	lw.ResponseData.status = code
	lw.wroteHeader = true
}

func (lw LoggableResponseWriter) Write(data []byte) (int, error) {
	if !lw.wroteHeader {
		lw.WriteHeader(http.StatusOK)
	}
	written, err := lw.ResponseWriter.Write(data)
	lw.ResponseData.size += written
	return written, err
}

// Log runs request and logs itself and returned response
func Log(next http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		uri := r.URL.Path
		method := r.Method
		rd := &ResponseData{
			status: http.StatusOK,
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Log.Warn("cannot read request body", zap.Error(err))
			http.Error(w, "cannot read request body", http.StatusBadRequest)
			return
		}
		// Восстанавливаем поток, чтобы обработчик тоже мог прочитать тело.
		r.Body = io.NopCloser(bytes.NewReader(body))

		// replace writer
		lw := &LoggableResponseWriter{
			ResponseWriter: w,
			ResponseData: rd,
		}

		// call request
		next.ServeHTTP(lw, r)

		since := time.Since(start)

		logger.Log.Info("REQUEST",
			zap.String("uri", uri),
			zap.String("method", method),
			zap.ByteString("body", body),
			zap.String("time", since.String()),
			zap.Any("headers", r.Header),
		)

		logger.Log.Info("RESPONSE",
			zap.Int("status", rd.status),
			zap.Int("size", rd.size),
		)
	}
	return http.HandlerFunc(logFn)
}
