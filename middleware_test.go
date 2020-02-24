package rest

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)

	ts := httptest.NewServer(Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		JsonResponse(w, []byte("OK"))
	})))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	fmt.Println(buf.String())
	require.NotEmpty(t, buf)
}
