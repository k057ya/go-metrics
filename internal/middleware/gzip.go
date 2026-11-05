package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type compressedWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

type Logger interface {
	Warnw(message string, params ...any)
	Errorw(message string, params ...any)
}

func (c compressedWriter) Header() http.Header {
	return c.w.Header()
}

func (c compressedWriter) Write(bytes []byte) (int, error) {
	return c.zw.Write(bytes)
}

func (c compressedWriter) WriteHeader(statusCode int) {
	c.w.Header().Set("Content-Encoding", "gzip")
	c.w.WriteHeader(statusCode)
}

func (c compressedWriter) Close() error {
	return c.zw.Close()
}

type compressedReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func (c compressedReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c compressedReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

func Compress(log Logger) func (http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return compress(next, log)
	}
}

func compress(next http.Handler, log Logger) http.Handler {
	compressFn := func(w http.ResponseWriter, r *http.Request) {

		ow := w

		contentType := r.Header.Get("Content-Type")
		if contentType == "application/json" || contentType == "text/html" || contentType == "" {

			// check if client supports gzip
			a := r.Header.Get("Accept-Encoding")
			if strings.Contains(a, "gzip") {
				// replace writer
				cw := &compressedWriter{w, gzip.NewWriter(w)}
				ow = cw
				defer func(){
					err := cw.Close()
					if err != nil {
						log.Warnw("error closing writer", err)
					}
				}()
			}

			e := r.Header.Get("Content-Encoding")
			if strings.Contains(e, "gzip") {
				cr, err := gzip.NewReader(r.Body)
				if err != nil {
					log.Warnw("Unable to read body as gzip",
						"error", err,
						"method", r.Method,
						"path", r.URL.Path,
					)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				r.Body = &compressedReader{r.Body, cr}
				defer func() {
					err := cr.Close()
					if err != nil {
						log.Warnw("error closing reader", err)
					}
				}()
			}

		}

		// call request
		next.ServeHTTP(ow, r)
	}

	return http.HandlerFunc(compressFn)
}
