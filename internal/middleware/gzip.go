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

func (c compressedWriter) Header() http.Header {
	return c.w.Header()
}

func (c compressedWriter) Write(bytes []byte) (int, error) {
	return c.zw.Write(bytes)
}

func (c compressedWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

func (c compressedWriter) Close() {
	c.zw.Close()
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

func Compress(next http.Handler) http.Handler {
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
				defer cw.Close()
			}

			e := r.Header.Get("Content-Encoding")
			if strings.Contains(e, "gzip") {
				cr, _ := gzip.NewReader(r.Body)
				r.Body = &compressedReader{r.Body, cr}
				defer cr.Close()
			}

		}

		// call request
		next.ServeHTTP(ow, r)
	}

	return http.HandlerFunc(compressFn)
}
