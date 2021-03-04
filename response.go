package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// HttpError - structure for http errors.
type HttpError struct {
	Err     string `json:"error"`
	Message string `json:"message,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}

var (
	ErrUnmarshal    = errors.New("UNMARSHAL_ERROR")
	ErrMissingField = errors.New("MISSING_FIELD")
	ErrNotFound     = errors.New("NOT_FOUND")
)

// Just to confirm Error interface.
func (e HttpError) Error() string {
	return e.Err
}

// RenderJSON sends data as json.
func RenderJSON(w http.ResponseWriter, code int, data interface{}) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)

	if data != nil {
		if err := enc.Encode(data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	w.WriteHeader(code)
	_, _ = w.Write(buf.Bytes())
}

// JsonResponse - write a response with application/json Content-Type header.
func JsonResponse(w http.ResponseWriter, data interface{}) {
	RenderJSON(w, http.StatusOK, data)
}

// OKResponse - write a OK response with application/json Content-Type header.
func OkResponse(w http.ResponseWriter) {
	RenderJSON(w, http.StatusOK, struct {
		OK bool `json:"ok"`
	}{
		OK: true,
	})
}

// JsonError - write a HttpError structure as response.
func ErrorResponse(w http.ResponseWriter, r *http.Request, code int, error error, msg string) {
	err := HttpError{
		Err:     http.StatusText(code),
		Message: msg,
		TraceID: r.Header.Get("Uber-Trace-Id"),
	}

	if error != nil {
		err.Err = error.Error()
	}
	err.Err = strings.ToUpper(err.Err)
	err.Err = strings.Replace(err.Err, " ", "_", -1)

	uri := r.URL.String()
	if qun, e := url.QueryUnescape(uri); e == nil {
		uri = qun
	}

	log.Printf("[DEBUG] %s - %s - %d (%s) - %s - %s", r.Method, uri, code, http.StatusText(code), err, msg)

	RenderJSON(w, code, err)
}

// NotFound - return a error page for not found
func NotFound(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("Not found."))
}
