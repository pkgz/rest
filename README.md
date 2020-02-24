# rest
[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/pkgz/logg)
[![Tests](https://img.shields.io/github/workflow/status/pkgz/rest/Code%20coverage)](https://github.com/pkgz/rest/actions)
[![codecov](https://img.shields.io/codecov/c/gh/pkgz/rest)](https://codecov.io/gh/pkgz/rest)

http/rest server, middleware and helpers.

## Installation
```bash
go get github.com/pkgz/rest
```

## Server
Create a simple http server with timeouts. 

## Middleware

### Logger
Log all requests with level DEBUG.

Log contains next parameters:

- Method
- requested url
- ip address (please use hide real user ip on prod)
- request duration
- response code

```bash
[DEBUG] GET - /test - 127.0.0.1 - 10.423µs - 200
```

## Helpers

### ReadBody
Read the body from request and trying to unmarshal to the provided struct.

### JsonResponse
Write a response with application/json Content-Type header.  
Except only bytes or struct.

### ErrorResponse
Makes error response easiest.   

```golang
JsonError(w, http.StatusBadRequest, err, "Missed value in request")
```

Error in response has the next structure:

```
type HttpError struct {
	Err     string `json:"error"`
	Message string `json:"message,omitempty"`
}
```

### NotFound
Handler for not found endpoint.  
Return next response:

```bash
Content-Type: text/plain
Status Code: 404 (Not Found)
Body: Not found.
```


## Licence
[MIT License](https://github.com/pkgz/rest/blob/master/LICENSE)