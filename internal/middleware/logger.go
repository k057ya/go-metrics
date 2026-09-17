package middleware

import (
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
}

func (lw LoggableResponseWriter) WriteHeader(code int) {
	lw.ResponseWriter.WriteHeader(code)
	lw.ResponseData.status = code
}

func (lw LoggableResponseWriter) Write(data []byte) (int, error) {
	written, err := lw.ResponseWriter.Write(data)
	if err != nil {
		return 0, err
	}
	lw.ResponseData.size += written

	return written, err
}

// Log runs request and logs itself and returned response
func Log(next http.Handler) http.Handler {

	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		uri := r.URL.Path
		method := r.Method
		rd := &ResponseData{}

		// replace writer
		lw := &LoggableResponseWriter{w, rd}

		// call request
		next.ServeHTTP(lw, r)

		since := time.Since(start)

		logger.Log.Info("REQUEST",
			zap.String("uri", uri),
			zap.String("method", method),
			zap.String("time", since.String()),
		)

		logger.Log.Info("RESPONSE",
			zap.Int("status", rd.status),
			zap.Int("size", rd.size),
		)
	}
	return http.HandlerFunc(logFn)
}
