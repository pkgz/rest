package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

type errReader int

var errBodyRead = errors.New("read body error")

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errBodyRead
}

func TestReadBody(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		var emptyStruct interface{}
		err := ReadBody(nil, emptyStruct)
		require.Error(t, err)
		require.True(t, errors.Is(ErrEmptyRequest, err))
	})

	t.Run("empty struct", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/something", nil)
		var emptyStruct interface{}
		err := ReadBody(r, emptyStruct)
		require.Error(t, err)
		require.True(t, errors.Is(ErrNotPointer, err))
	})

	t.Run("read error", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/something", errReader(0))
		var emptyStruct interface{}
		err := ReadBody(r, &emptyStruct)
		require.Error(t, err)
		require.Equal(t, errBodyRead, err)
	})

	t.Run("unmarshal struct", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/something", nil)
		var emptyStruct interface{}
		err := ReadBody(r, &emptyStruct)
		require.Error(t, err)
		require.Equal(t, "unexpected end of JSON input", err.Error())
	})

	t.Run("good request", func(t *testing.T) {
		requestBody := struct {
			Name string
		}{
			Name: "test",
		}
		b, err := json.Marshal(requestBody)
		require.NoError(t, err)

		r := httptest.NewRequest(http.MethodPost, "/something", bytes.NewBuffer(b))
		var emptyStruct struct {
			Name string
		}
		require.NoError(t, ReadBody(r, &emptyStruct))

		require.Equal(t, requestBody.Name, emptyStruct.Name)
	})
}

func TestGetAddr(t *testing.T) {
	t.Run("all empty IP sources", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ""

		req.Header.Set("CF-Connecting-IP", "")
		req.Header.Set("X-Forwarded-For", "")
		req.Header.Set("X-Real-Ip", "")

		addr := GetAddr(req)
		require.Equal(t, "", addr)
	})

	t.Run("IPv4 address from RemoteAddr", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"

		addr := GetAddr(req)
		require.Equal(t, "192.168.1.1", addr)
	})

	t.Run("IPv6 address from RemoteAddr", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "[2001:db8::1]:12345"

		addr := GetAddr(req)
		require.Equal(t, "[2001:db8::1]", addr)
	})

	t.Run("CF-Connecting-IP header priority", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("CF-Connecting-IP", "203.0.113.195")
		req.Header.Set("X-Forwarded-For", "198.51.100.23")
		req.Header.Set("X-Real-Ip", "192.0.2.50")

		addr := GetAddr(req)
		require.Equal(t, "203.0.113.195", addr)
	})

	t.Run("X-Forwarded-For fallback", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ""
		req.Header.Set("X-Forwarded-For", "198.51.100.23")
		req.Header.Set("X-Real-Ip", "192.0.2.50")

		addr := GetAddr(req)
		require.Equal(t, "198.51.100.23", addr)
	})

	t.Run("X-Real-Ip final fallback", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ""
		req.Header.Set("X-Real-Ip", "192.0.2.50")

		addr := GetAddr(req)
		require.Equal(t, "192.0.2.50", addr)
	})

	t.Run("X-Forwarded-For with multiple IPs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ""
		req.Header.Set("X-Forwarded-For", "203.0.113.195, 198.51.100.23, 192.0.2.50")

		addr := GetAddr(req)
		require.Equal(t, "203.0.113.195", addr)
	})

	t.Run("IPv6 in X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ""
		req.Header.Set("X-Forwarded-For", "2001:db8:85a3::8a2e:370:7334")

		addr := GetAddr(req)
		require.Equal(t, "2001:db8:85a3::8a2e:370:7334", addr)
	})

	t.Run("malformed IP address", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "invalid-ip-format"

		addr := GetAddr(req)
		require.Equal(t, "invalid-ip-format", addr)
	})

	t.Run("empty IPv4 address with port only", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ":2102"

		addr := GetAddr(req)
		require.Equal(t, "", addr)
	})

	t.Run("IPv6 localhost with port", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "[::]:2102"

		addr := GetAddr(req)
		require.Equal(t, "[::]", addr)
	})

	t.Run("IPv6 localhost alternative form with port", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "[::1]:2102"

		addr := GetAddr(req)
		require.Equal(t, "[::1]", addr)
	})
}
