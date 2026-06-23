package rest

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/jwtauth/v5"
	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"
)

func newRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

// jwksServer serves a JWKS document at /.well-known/jwks.json built from the provided keys.
func jwksServer(t *testing.T, jwks jose.JSONWebKeySet) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})
	return httptest.NewServer(mux)
}

func publicJWKS(key *rsa.PrivateKey, keyID string) jose.JSONWebKeySet {
	return jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key:       &key.PublicKey,
		KeyID:     keyID,
		Algorithm: "RS256",
		Use:       "sig",
	}}}
}

// signToken builds a signed RS256 token string with the given claims.
func signToken(t *testing.T, key *rsa.PrivateKey, claims map[string]interface{}) string {
	t.Helper()
	signer := jwtauth.New("RS256", key, &key.PublicKey)
	_, tokenString, err := signer.Encode(claims)
	require.NoError(t, err)
	return tokenString
}

func TestIntersect(t *testing.T) {
	testCases := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{"common element", []string{"read", "write"}, []string{"write", "admin"}, true},
		{"no common element", []string{"read"}, []string{"admin"}, false},
		{"empty a", nil, []string{"admin"}, false},
		{"empty b", []string{"read"}, nil, false},
		{"both empty", nil, nil, false},
		{"identical", []string{"read"}, []string{"read"}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, intersect(tc.a, tc.b))
		})
	}
}

func TestGetPublicKey(t *testing.T) {
	t.Run("success returns first key", func(t *testing.T) {
		key := newRSAKey(t)
		ts := jwksServer(t, publicJWKS(key, "key-1"))
		defer ts.Close()

		pub, err := getPublicKey(t.Context(), ts.URL)
		require.NoError(t, err)
		require.True(t, pub.Equal(&key.PublicKey))
	})

	t.Run("selects key matching JWKS_KEY_ID", func(t *testing.T) {
		key1 := newRSAKey(t)
		key2 := newRSAKey(t)
		jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
			{Key: &key1.PublicKey, KeyID: "key-1", Algorithm: "RS256", Use: "sig"},
			{Key: &key2.PublicKey, KeyID: "key-2", Algorithm: "RS256", Use: "sig"},
		}}
		ts := jwksServer(t, jwks)
		defer ts.Close()

		t.Setenv("JWKS_KEY_ID", "key-2")

		pub, err := getPublicKey(t.Context(), ts.URL)
		require.NoError(t, err)
		require.True(t, pub.Equal(&key2.PublicKey))
	})

	t.Run("error when JWKS_KEY_ID not found", func(t *testing.T) {
		key := newRSAKey(t)
		ts := jwksServer(t, publicJWKS(key, "key-1"))
		defer ts.Close()

		t.Setenv("JWKS_KEY_ID", "missing")

		_, err := getPublicKey(t.Context(), ts.URL)
		require.Error(t, err)
		require.Contains(t, err.Error(), "public JWK not found")
	})

	t.Run("error on non-200 status", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		_, err := getPublicKey(t.Context(), ts.URL)
		require.Error(t, err)
		require.Contains(t, err.Error(), "wrong status code")
	})

	t.Run("error on invalid JSON", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not-json"))
		}))
		defer ts.Close()

		_, err := getPublicKey(t.Context(), ts.URL)
		require.Error(t, err)
	})

	t.Run("error on empty key set", func(t *testing.T) {
		ts := jwksServer(t, jose.JSONWebKeySet{})
		defer ts.Close()

		_, err := getPublicKey(t.Context(), ts.URL)
		require.Error(t, err)
		require.Contains(t, err.Error(), "public JWKS not found")
	})

	t.Run("error when JWK is not a public key", func(t *testing.T) {
		key := newRSAKey(t)
		jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
			Key:       key, // full private key
			KeyID:     "key-1",
			Algorithm: "RS256",
			Use:       "sig",
		}}}
		ts := jwksServer(t, jwks)
		defer ts.Close()

		_, err := getPublicKey(t.Context(), ts.URL)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not public")
	})

	t.Run("error on unreachable host", func(t *testing.T) {
		_, err := getPublicKey(t.Context(), "http://127.0.0.1:0")
		require.Error(t, err)
	})
}

func TestParseToken(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("valid token sets context and passes", func(t *testing.T) {
		key := newRSAKey(t)
		ts := jwksServer(t, publicJWKS(key, "key-1"))
		defer ts.Close()
		t.Setenv("AUTH_HOST", ts.URL)

		var gotSub, gotScope interface{}
		handler := ParseToken()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotSub = r.Context().Value("sub")
			gotScope = r.Context().Value("scope")
			w.WriteHeader(http.StatusOK)
		}))

		token := signToken(t, key, map[string]interface{}{"sub": "user-42", "scope": "read,write"})
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "BEARER "+token)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)
		require.Equal(t, "user-42", gotSub)
		require.Equal(t, "read,write", gotScope)
	})

	t.Run("missing token returns 401", func(t *testing.T) {
		key := newRSAKey(t)
		ts := jwksServer(t, publicJWKS(key, "key-1"))
		defer ts.Close()
		t.Setenv("AUTH_HOST", ts.URL)

		handler := ParseToken()(okHandler)
		req := httptest.NewRequest("GET", "/protected", nil)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		require.Equal(t, http.StatusUnauthorized, resp.Code)
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		key := newRSAKey(t)
		ts := jwksServer(t, publicJWKS(key, "key-1"))
		defer ts.Close()
		t.Setenv("AUTH_HOST", ts.URL)

		// token signed by a different key fails verification
		other := newRSAKey(t)
		token := signToken(t, other, map[string]interface{}{"sub": "user", "scope": "read"})

		handler := ParseToken()(okHandler)
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "BEARER "+token)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		require.Equal(t, http.StatusUnauthorized, resp.Code)
	})

	t.Run("missing scope claim returns 403", func(t *testing.T) {
		key := newRSAKey(t)
		ts := jwksServer(t, publicJWKS(key, "key-1"))
		defer ts.Close()
		t.Setenv("AUTH_HOST", ts.URL)

		token := signToken(t, key, map[string]interface{}{"sub": "user"})

		handler := ParseToken()(okHandler)
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "BEARER "+token)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		require.Equal(t, http.StatusForbidden, resp.Code)
	})

	t.Run("scope mismatch returns 403", func(t *testing.T) {
		key := newRSAKey(t)
		ts := jwksServer(t, publicJWKS(key, "key-1"))
		defer ts.Close()
		t.Setenv("AUTH_HOST", ts.URL)

		token := signToken(t, key, map[string]interface{}{"sub": "user", "scope": "read"})

		handler := ParseToken("admin")(okHandler)
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "BEARER "+token)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		require.Equal(t, http.StatusForbidden, resp.Code)
	})

	t.Run("matching required scope passes", func(t *testing.T) {
		key := newRSAKey(t)
		ts := jwksServer(t, publicJWKS(key, "key-1"))
		defer ts.Close()
		t.Setenv("AUTH_HOST", ts.URL)

		token := signToken(t, key, map[string]interface{}{"sub": "user", "scope": "read,admin"})

		handler := ParseToken("admin")(okHandler)
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "BEARER "+token)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("token from query parameter", func(t *testing.T) {
		key := newRSAKey(t)
		ts := jwksServer(t, publicJWKS(key, "key-1"))
		defer ts.Close()
		t.Setenv("AUTH_HOST", ts.URL)

		token := signToken(t, key, map[string]interface{}{"sub": "user", "scope": "read"})

		handler := ParseToken()(okHandler)
		req := httptest.NewRequest("GET", "/protected?jwt="+token, nil)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)
	})
}
