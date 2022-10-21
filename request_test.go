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
