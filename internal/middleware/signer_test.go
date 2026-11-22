package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/k057ya/go-metrics/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCheckAndSign(t *testing.T) {
	const key = "test-key"
	const body = `{"id":"Alloc","type":"gauge","value":12.5}`
	tests := []struct {
		name       string
		signature  string
		wantStatus int
	}{
		{name: "unsigned request", wantStatus: http.StatusCreated},
		{name: "valid signature", signature: service.GenSignatureBase64([]byte(body), key), wantStatus: http.StatusCreated},
		{name: "wrong key", signature: service.GenSignatureBase64([]byte(body), "wrong-key"), wantStatus: http.StatusBadRequest},
		{name: "changed body", signature: service.GenSignatureBase64([]byte(body+" "), key), wantStatus: http.StatusBadRequest},
		{name: "malformed signature", signature: "not-base64", wantStatus: http.StatusBadRequest},
	}

	for _, test := range tests {
		log := zap.NewNop().Sugar()
		t.Run(test.name, func(t *testing.T) {
			called := false
			handler := CheckAndSign(key, log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				gotBody, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, body, string(gotBody))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, err = io.WriteString(w, `{"ok":`)
				require.NoError(t, err)
				_, err = io.WriteString(w, `true}`)
				require.NoError(t, err)
			}))

			request := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(body))
			if test.signature != "" {
				request.Header.Set("HashSHA256", test.signature)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			response := recorder.Result()
			t.Cleanup(func() { require.NoError(t, response.Body.Close()) })

			assert.Equal(t, test.wantStatus, response.StatusCode)
			assert.Equal(t, test.wantStatus == http.StatusCreated, called)
			if !called {
				assert.Equal(t, "invalid signature\n", recorder.Body.String())
				return
			}

			assert.Equal(t, `{"ok":true}`, recorder.Body.String())
			assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
			assert.Equal(t, service.GenSignatureBase64(recorder.Body.Bytes(), key), response.Header.Get("HashSHA256"))
		})
	}
}

func TestCheckAndSignResponseStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantStatus int
	}{
		{name: "implicit OK", body: "response", wantStatus: http.StatusOK},
		{name: "explicit status", status: http.StatusAccepted, body: "response", wantStatus: http.StatusAccepted},
		{name: "no content", status: http.StatusNoContent, wantStatus: http.StatusNoContent},
	}
	log := zap.NewNop().Sugar()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := CheckAndSign("test-key", log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.status != 0 {
					w.WriteHeader(test.status)
					w.WriteHeader(http.StatusInternalServerError)
				}
				if test.body != "" {
					_, err := io.WriteString(w, test.body)
					require.NoError(t, err)
				}
			}))
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			response := recorder.Result()
			t.Cleanup(func() { require.NoError(t, response.Body.Close()) })

			assert.Equal(t, test.wantStatus, response.StatusCode)
			assert.Equal(t, test.body, recorder.Body.String())
			assert.Equal(t, service.GenSignatureBase64([]byte(test.body), "test-key"), response.Header.Get("HashSHA256"))
		})
	}
}
