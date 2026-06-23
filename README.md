# rest
[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/pkgz/rest)
[![Tests](https://img.shields.io/github/workflow/status/pkgz/rest/Code%20coverage)](https://github.com/pkgz/rest/actions)
[![codecov](https://img.shields.io/codecov/c/gh/pkgz/rest)](https://codecov.io/gh/pkgz/rest)

http/rest server, middleware and helpers.

## Installation
```bash
go get github.com/pkgz/rest
```

## Server
Create an http server with sensible default timeouts.

```golang
srv := rest.NewServer(8080)
if err := srv.Run(router); err != nil {
    log.Fatal(err)
}
```

Timeouts default when left at zero and can be overridden on the `Server` struct:

| Field               | Default      |
|---------------------|--------------|
| `ReadHeaderTimeout` | `10s`        |
| `ReadTimeout`       | `30s`        |
| `WriteTimeout`      | `30s`        |
| `IdleTimeout`       | `60s`        |

When no router is provided, a default one is used exposing `/ping`, `/liveness`
and a `/readiness` probe. Set `SSL` to serve HTTPS (with optional HTTP→HTTPS
redirect). Call `Shutdown()` for a graceful stop.

## Middleware

### Logger
Log all requests with level DEBUG.

The log line contains:

- Method
- requested url (sensitive query params are masked, see below)
- client ip address (see `GetAddr`)
- response code
- request duration

```bash
[DEBUG] GET - /test - 127.0.0.1 - 200 - 10.423µs
```

Sensitive query parameters (`jwt`, `token`, `access_token`, `api_key`) are masked
as `***` before logging, so secrets passed in the URL never reach the logs.

Requests to known health-check paths (`rest.HealthPaths`: `/ping`, `/liveness`,
`/readiness`) are not logged, to keep probe noise out of the logs. The list is a
package variable you can extend.

### Readiness
Middleware for a readiness probe. Returns `503 Service Unavailable` until the
provided `*atomic.Value` holds `true`.

```golang
isReady := &atomic.Value{}
isReady.Store(false)
router.Use(rest.Readiness("/readiness", isReady))
```

### ParseToken
Verifies the request JWT (`RS256`) against the public key published at
`${AUTH_HOST}/.well-known/jwks.json`, optionally enforcing scopes, and adds the
`sub` and `scope` claims to the request context.

```golang
router.Use(rest.ParseToken("admin", "user"))
```

Requires the `AUTH_HOST` environment variable. `JWKS_KEY_ID` may be set to select
a specific key from the key set.

## Helpers

### ReadBody
Read the body from a request and unmarshal it into the provided struct pointer.
Reads are capped at `rest.MaxBodyBytes` (default 1 MiB) to bound memory use.

### GetAddr
Resolve the client address. Proxy-supplied headers take precedence (when present)
over `RemoteAddr`, in the order: `CF-Connecting-IP`, `X-Forwarded-For` (first
entry), `X-Real-Ip`, then `RemoteAddr`. These headers are client-spoofable, so
only rely on them behind a trusted proxy.

### JsonResponse
Write a response with `application/json` Content-Type header.

### TextResponse
Write a plain string response with `text/html` Content-Type header.

### OkResponse
Write `{"ok":true}` with an `application/json` Content-Type header.

### ErrorResponse
Write an error response.

```golang
rest.ErrorResponse(w, r, http.StatusBadRequest, err, "Missed value in request")
```

The error response has the following structure:

```golang
type HttpError struct {
	Err     string `json:"error"`
	Message string `json:"message,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}
```

`TraceID` is populated from the request's `Uber-Trace-Id` header when present.

### NotFound
Handler for a not found endpoint. Returns:

```bash
Content-Type: text/plain
Status Code: 404 (Not Found)
Body: Not found.
```

## Licence
[MIT License](https://github.com/pkgz/rest/blob/master/LICENSE)
</content>
</invoke>
