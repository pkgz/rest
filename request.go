package rest

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
)

var ErrEmptyRequest = errors.New("empty request")
var ErrNotPointer = errors.New("not pointer provided")

// ReadBody - read body from request and trying to unmarshal to provided struct
func ReadBody(w http.ResponseWriter, r *http.Request, str interface{}) error {
	if r == nil {
		return ErrEmptyRequest
	}
	if reflect.ValueOf(str).Kind() != reflect.Ptr {
		return ErrNotPointer
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil && err != io.EOF {
		return err
	}

	err = json.Unmarshal(body, str)
	if err != nil {
		return err
	}

	return nil
}
