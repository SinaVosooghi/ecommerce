package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sinavosooghi/ecommerce/services/cart-service/internal/logging"
	"github.com/stretchr/testify/assert"
)

func TestRecovery_ReturnsInternalError(t *testing.T) {
	logger := logging.New(logging.Config{Level: "error", Output: io.Discard})
	handler := Recovery(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "INTERNAL_ERROR")
}

func TestRecovery_RepanicsOnAbortHandler(t *testing.T) {
	logger := logging.New(logging.Config{Level: "error", Output: io.Discard})
	handler := Recovery(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	})
}
