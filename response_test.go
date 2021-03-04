package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type custom struct {
	test string
}

func (c *custom) MarshalJSON() ([]byte, error) {
	return nil, errors.New("test")
}

func TestJsonError(t *testing.T) {
	traceID := fmt.Sprintf("%d", time.Now().Unix())

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/empty_error" {
			ErrorResponse(w, r, http.StatusInternalServerError, nil, "test")
			return
		} else if r.URL.Path == "/trace_id" {
			r.Header.Set("Uber-Trace-Id", traceID)
			ErrorResponse(w, r, http.StatusBadRequest, nil, "trace-id-test")
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

		var response HttpError
		require.NoError(t, json.Unmarshal(body, &response))

		require.Equal(t, "ERR_TEST", response.Err)
		require.Equal(t, "test", response.Message)
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

		var response HttpError
		require.NoError(t, json.Unmarshal(body, &response))

		require.Equal(t, "INTERNAL_SERVER_ERROR", response.Err)
		require.Equal(t, "test", response.Message)
		require.Empty(t, response.TraceID)
	})

	t.Run("trace id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://test/trace_id", nil)
		w := httptest.NewRecorder()
		handler(w, req)
		resp := w.Result()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		var response HttpError
		require.NoError(t, json.Unmarshal(body, &response))

		require.Equal(t, "BAD_REQUEST", response.Err)
		require.Equal(t, "trace-id-test", response.Message)
		require.Equal(t, traceID, response.TraceID)
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
		} else if r.URL.Path == "/multiply-header-set" {
			JsonResponse(w, &custom{
				test: "test",
			})
			return
		}
		OkResponse(w)
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

	t.Run("multiply header set", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://test/multiply-header-set", nil)
		w := httptest.NewRecorder()
		handler(w, req)
		resp := w.Result()

		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, "json: error calling MarshalJSON for type *rest.custom: test\n", string(body))
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
