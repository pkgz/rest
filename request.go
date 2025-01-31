package rest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
)

var ErrEmptyRequest = errors.New("empty request")
var ErrNotPointer = errors.New("not pointer provided")

// ReadBody - read body from request and trying to unmarshal to provided struct
func ReadBody(r *http.Request, str interface{}) error {
	if r == nil {
		return ErrEmptyRequest
	}
	if reflect.ValueOf(str).Kind() != reflect.Ptr {
		return ErrNotPointer
	}

	body, err := io.ReadAll(r.Body)
	if err != nil && err != io.EOF {
		return err
	}
	defer func() { _ = r.Body.Close() }()

	if err = json.Unmarshal(body, str); err != nil {
		return err
	}

	return nil
}

func GetAddr(r *http.Request) string {
	addr := r.RemoteAddr
	if CFAddr := r.Header.Get("CF-Connecting-IP"); CFAddr != "" {
		addr = CFAddr
	}
	if addr == "" {
		addr = r.Header.Get("X-Forwarded-For")
	}
	if addr == "" {
		addr = r.Header.Get("X-Real-Ip")
	}

	if strings.Count(addr, ":") == 1 {
		addr = strings.Split(addr, ":")[0]
	}

	return addr
}
