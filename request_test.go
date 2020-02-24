package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/require"
	"io/ioutil"
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
		w := httptest.NewRecorder()
		var emptyStruct interface{}
		err := ReadBody(w, nil, emptyStruct)
		require.Error(t, err)
		require.True(t, errors.Is(ErrEmptyRequest, err))
	})

	t.Run("empty struct", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/something", nil)
		var emptyStruct interface{}
		err := ReadBody(w, r, emptyStruct)
		require.Error(t, err)
		require.True(t, errors.Is(ErrNotPointer, err))
	})

	t.Run("read error", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/something", errReader(0))
		var emptyStruct interface{}
		err := ReadBody(w, r, &emptyStruct)
		require.Error(t, err)

		resp := w.Result()
		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var respError HttpError
		err = json.Unmarshal(body, &respError)
		require.NoError(t, err)

		require.NotEmpty(t, respError.Err)
	})

	t.Run("unmarshal struct", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/something", nil)
		var emptyStruct interface{}
		err := ReadBody(w, r, &emptyStruct)
		require.Error(t, err)

		resp := w.Result()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var respError HttpError
		err = json.Unmarshal(body, &respError)
		require.NoError(t, err)

		require.NotEmpty(t, respError.Err)
	})

	t.Run("good request", func(t *testing.T) {
		requestBody := struct {
			Name string
		}{
			Name: "test",
		}
		b, err := json.Marshal(requestBody)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/something", bytes.NewBuffer(b))
		var emptyStruct struct {
			Name string
		}
		err = ReadBody(w, r, &emptyStruct)
		require.NoError(t, err)

		require.Equal(t, requestBody.Name, emptyStruct.Name)
	})
}
