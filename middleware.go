package rest

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/go-jose/go-jose/v4"
)

var (
	sensitiveQueryParams = []string{"jwt", "token", "access_token", "api_key"}

	// HealthPaths - known health-check paths silenced.
	HealthPaths = []string{"/ping", "/liveness", "/readiness", "/health"}
)

// Logger - log all requests
func Logger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.URL != nil && slices.Contains(HealthPaths, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		ww := middleware.NewWrapResponseWriter(w, 1)
		start := time.Now()

		defer func() {
			statusCode := ww.Status()
			if statusCode == 0 {
				statusCode = 200
			}

			var uri string
			if r.URL != nil {
				uri = r.URL.String()
				uri = SanitizeURL(uri)
				if qun, e := url.QueryUnescape(uri); e == nil {
					uri = qun
				}
			} else {
				uri = "<nil>"
			}

			duration := time.Since(start)
			log.Printf("[DEBUG] %s - %s - %s - %v - %v", r.Method, uri, GetAddr(r), statusCode, duration)
		}()

		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}

// Readiness - middleware for the readiness probe
func Readiness(endpoint string, isReady *atomic.Value) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" && strings.EqualFold(r.URL.Path, endpoint) {
				if isReady == nil || !isReady.Load().(bool) {
					ErrorResponse(w, r, http.StatusServiceUnavailable, nil, "")
					return
				}

				OkResponse(w)
				return
			}

			h.ServeHTTP(w, r)
		})
	}
}

// SanitizeURL replaces sensitive query parameter values with asterisks in the given URL
func SanitizeURL(uri string) string {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	query := parsedURL.Query()

	var masked []string
	for _, param := range sensitiveQueryParams {
		if query.Get(param) != "" {
			query.Del(param)
			masked = append(masked, param+"=***")
		}
	}

	if len(masked) > 0 {
		encodedQuery := query.Encode()
		if encodedQuery != "" {
			parsedURL.RawQuery = encodedQuery + "&" + strings.Join(masked, "&")
		} else {
			parsedURL.RawQuery = strings.Join(masked, "&")
		}
	}

	return parsedURL.String()
}

// ParseToken - parsing the JWT token in the request. Add a sub and scope to the request context.
func ParseToken(scope ...string) func(http.Handler) http.Handler {
	authHost := os.Getenv("AUTH_HOST")
	if authHost == "" {
		log.Fatal("AUTH_HOST environment is required")
		return nil
	}

	publicKey, err := getPublicKey(context.Background(), authHost)
	if err != nil {
		log.Fatalf("get public key: %v", err)
		return nil
	}

	ja := jwtauth.New("RS256", nil, publicKey)

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			token, err := jwtauth.VerifyRequest(ja, r, jwtauth.TokenFromQuery, jwtauth.TokenFromHeader, jwtauth.TokenFromCookie)
			if err != nil {
				if errors.Is(err, jwtauth.ErrNoTokenFound) {
					ErrorResponse(w, r, http.StatusUnauthorized, nil, "missing jwt token")
					return
				}
				log.Printf("[ERROR] verify jwt token: %v", err)
				ErrorResponse(w, r, http.StatusUnauthorized, nil, "verify jwt token")
				return
			}

			var scopeClaim string
			if err := token.Get("scope", &scopeClaim); err != nil {
				log.Printf("[ERROR] missing or invalid scope claim: %v", err)
				ErrorResponse(w, r, http.StatusForbidden, nil, "invalid token")
				return
			}

			if scope != nil {
				userScopes := strings.Split(scopeClaim, ",")
				if !intersect(userScopes, scope) {
					log.Printf("[ERROR] user scopes: %v, required scopes: %v", userScopes, scope)
					ErrorResponse(w, r, http.StatusForbidden, nil, "not allowed")
					return
				}
			}

			sub, ok := token.Subject()
			if !ok {
				log.Printf("[ERROR] missing or invalid sub claim")
				ErrorResponse(w, r, http.StatusForbidden, nil, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), "sub", sub)
			ctx = context.WithValue(ctx, "scope", scopeClaim)

			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}
func getPublicKey(ctx context.Context, host string) (*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/.well-known/jwks.json", host), nil)
	if err != nil {
		return nil, err
	}

	client := http.Client{
		Timeout: time.Second * 3,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("wrong status code: %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jwks = jose.JSONWebKeySet{}
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, err
	}

	publicJWKS := jwks.Keys
	if len(publicJWKS) == 0 {
		return nil, errors.New("public JWKS not found")
	}

	publicJWK := publicJWKS[0]
	if keyID := os.Getenv("JWKS_KEY_ID"); keyID != "" {
		found := false

		for _, key := range publicJWKS {
			if key.KeyID == keyID {
				publicJWK = key
				found = true
				break
			}
		}

		if !found {
			return nil, errors.New("public JWK not found")
		}
	}

	if !publicJWK.IsPublic() {
		return nil, errors.New("JWK is not public key")
	}

	return publicJWK.Key.(*rsa.PublicKey), nil
}
func intersect(a []string, b []string) bool {
	for _, itemA := range a {
		for _, itemB := range b {
			if itemA == itemB {
				return true
			}
		}
	}
	return false
}
