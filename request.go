package rest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
)

var (
	ErrEmptyRequest = errors.New("empty request")
	ErrNotPointer   = errors.New("not pointer provided")

	// MaxBodyBytes - maximum number of bytes ReadBody will read from a request body
	MaxBodyBytes int64 = 1 << 20 // 1 MiB
)

// ReadBody - read body from request and trying to unmarshal to provided struct
func ReadBody(r *http.Request, str any) error {
	if r == nil {
		return ErrEmptyRequest
	}
	if reflect.ValueOf(str).Kind() != reflect.Ptr {
		return ErrNotPointer
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, MaxBodyBytes))
	if err != nil && err != io.EOF {
		return err
	}
	defer func() { _ = r.Body.Close() }()

	if err = json.Unmarshal(body, str); err != nil {
		return err
	}

	return nil
}

// GetAddr - get client address from request.
//
// Proxy-supplied headers take precedence (when present) over r.RemoteAddr, in the
// order: CF-Connecting-IP, X-Forwarded-For (first entry), X-Real-Ip, then RemoteAddr.
// These headers are client-spoofable, so only rely on them behind a trusted proxy.
func GetAddr(r *http.Request) string {
	var addr string
	switch {
	case r.Header.Get("CF-Connecting-IP") != "":
		addr = r.Header.Get("CF-Connecting-IP")
	case r.Header.Get("X-Forwarded-For") != "":
		// X-Forwarded-For may be a comma-separated list; the first entry is the originating client.
		addr = r.Header.Get("X-Forwarded-For")
		if i := strings.Index(addr, ","); i >= 0 {
			addr = addr[:i]
		}
		addr = strings.TrimSpace(addr)
	case r.Header.Get("X-Real-Ip") != "":
		addr = r.Header.Get("X-Real-Ip")
	default:
		addr = r.RemoteAddr
	}

	if strings.HasPrefix(addr, "[") && strings.Contains(addr, "]:") {
		if i := strings.LastIndex(addr, "]:"); i >= 0 {
			addr = addr[:i+1]
		}
	} else if strings.Count(addr, ":") == 1 {
		addr = strings.Split(addr, ":")[0]
	}

	return addr
}
