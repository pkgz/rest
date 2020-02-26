package rest

import (
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJsonError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/empty_error" {
			ErrorResponse(w, r, http.StatusInternalServerError, nil, "test")
			return
		}
		ErrorResponse(w, r, http.StatusUnauthorized, errors.New("err test"), "test")
		return
	}

	t.Run("error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://test", nil)
		w := httptest.NewRecorder()
		handler(w, req)
		resp := w.Result()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var respError HttpError
		err = json.Unmarshal(body, &respError)
		require.NoError(t, err)

		require.Equal(t, "err test", respError.Err)
		require.Equal(t, "test", respError.Message)
	})

	t.Run("empty error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://test/empty_error", nil)
		w := httptest.NewRecorder()
		handler(w, req)
		resp := w.Result()
		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var respError HttpError
		err = json.Unmarshal(body, &respError)
		require.NoError(t, err)

		require.Equal(t, http.StatusText(http.StatusInternalServerError), respError.Err)
		require.Equal(t, "test", respError.Message)
	})
}

func TestJsonResponse(t *testing.T) {
	response := struct {
		OK bool `json:"ok"`
	}{
		OK: true,
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/empty" {
			JsonResponse(w, nil)
			return
		} else if r.URL.Path == "/string" {
			JsonResponse(w, []byte("OK"))
			return
		}
		JsonResponse(w, response)
	}

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://test/empty", nil)
		w := httptest.NewRecorder()
		handler(w, req)
		resp := w.Result()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Empty(t, body)
	})

	t.Run("struct", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://test", nil)
		w := httptest.NewRecorder()
		handler(w, req)
		resp := w.Result()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		b, err := json.Marshal(response)
		require.NoError(t, err)

		require.Equal(t, append(b, '\n'), body)
	})
}

func TestNotFound(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/exist" {
			w.WriteHeader(http.StatusOK)
			return
		}
		NotFound(w, r)
	}

	t.Run("existing page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://test/exist", nil)
		w := httptest.NewRecorder()
		handler(w, req)
		resp := w.Result()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://test/nope", nil)
		w := httptest.NewRecorder()
		handler(w, req)
		resp := w.Result()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestHttpError_Error(t *testing.T) {
	err := HttpError{
		Err:     "err test",
		Message: "message test",
	}

	require.Equal(t, "err test", err.Error())
}
