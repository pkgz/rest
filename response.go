package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// HttpError - structure for http errors.
type HttpError struct {
	Err     string `json:"error"`
	Message string `json:"message,omitempty"`
}

// Just to confirm Error interface.
func (e HttpError) Error() string {
	return e.Err
}

// RenderJSON sends data as json.
func RenderJSON(w http.ResponseWriter, data interface{}) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)

	if data != nil {
		if err := enc.Encode(data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	_, _ = w.Write(buf.Bytes())
}

// JsonResponse - write a response with application/json Content-Type header.
func JsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	RenderJSON(w, data)
}

// JsonError - write a HttpError structure as response.
func ErrorResponse(w http.ResponseWriter, code int, error error, msg string) {
	err := HttpError{
		Err:     http.StatusText(code),
		Message: msg,
	}

	if error != nil {
		err.Err = error.Error()
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	RenderJSON(w, err)
}

// NotFound - return a error page for not found
func NotFound(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("Not found."))
}