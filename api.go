// Package mixer contains the HTTP multiplexer based on a similar radix tree data structure.
package mixer

import (
	"context"
	"errors"
	"net/http"
)

type (
	// PathParams represents a map of path params that will store by index.
	PathParams map[int]any

	// ServeMuxError decorates all possible external errors to one kind.
	ServeMuxError struct {
		err     error
		method  string
		pattern string
	}

	// ServeMux is an HTTP request multiplexer.
	ServeMux struct {
		tree       *tree
		converters map[string]*convert
	}
)

const (
	// PathParamsCtxKey is a context key for using in context.Value.
	PathParamsCtxKey contextKey = "path-params"
)

var (
	// ErrMethod is the error if try set handler for the wrong method inside ServeMux.
	ErrMethod = errors.New("invalid method")

	// ErrHandler is the error if try to set nil handler inside ServeMux.
	ErrHandler = errors.New("nil handler")

	// ErrPattern is the error if we get an invalid URL pattern.
	ErrPattern = errors.New("invalid pattern")

	// ErrDuplicate is the error if we get multiple handlers for one method.
	ErrDuplicate = errors.New("duplicate handler")

	// ErrNotFound is the error if handler for combination method and pattern not exist.
	ErrNotFound = errors.New("not found")

	// ErrPathParam signals that typed path param is invalid.
	ErrPathParam = errors.New("invalid path param")

	// ErrMultiplePathParam signals that insert tried to perform
	// operation on invalid rule: for more see definition of node.
	ErrMultiplePathParam = errors.New("multiple types for path param")
)

// Error implements the error's Error.
func (e *ServeMuxError) Error() string {
	return "mixer: handler (" + e.method + ") " + e.pattern + " error: " + e.err.Error()
}

// Unwrap implements the error's Unwrap.
func (e *ServeMuxError) Unwrap() error {
	return e.err
}

// GetPathParams returns the path params registered in r.Context() or nil otherwise.
func GetPathParams(r *http.Request) PathParams {
	params, ok := r.Context().Value(PathParamsCtxKey).(PathParams)

	if !ok {
		return nil
	}

	return params
}

// Handler returns the handler to use for the given request.
func (mux *ServeMux) Handler(req *http.Request) (http.Handler, error) {
	url := req.URL.EscapedPath()
	parts, _ := splitURL(url)

	idx := 0
	node := mux.tree.root
	params := make(PathParams)

	for _, part := range parts {
		child, found := node.Children[part]
		if found {
			node = child

			continue
		}

		child, found = node.Children[typeToken]
		if !found {
			return nil, notFoundError(req.Method, url)
		}

		val, err := (*child.conv)(part)
		if err != nil {
			return nil, notFoundError(req.Method, url)
		}

		node = child
		params[idx] = val
		idx++
	}

	if node.Methods == nil {
		return nil, notFoundError(req.Method, url)
	}

	handler := node.Methods[req.Method]
	if handler == nil {
		return nil, notFoundError(req.Method, url)
	}

	if len(params) != 0 {
		*req = *req.WithContext(context.WithValue(req.Context(), PathParamsCtxKey, params))
	}

	return handler, nil
}

// Handle registers the handler for the given method and pattern.
// Because it is an initialization moment will be panics in any error.
func (mux *ServeMux) Handle(method, pattern string, handler http.Handler) {
	switch method {
	case
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace:
	default:
		panic(methodError(method, pattern))
	}

	if handler == nil {
		panic(handlerError(method, pattern))
	}

	parts, err := splitURL(pattern)
	if err != nil {
		panic(patternError(method, pattern))
	}

	methods, err := mux.insert(parts)
	if err != nil {
		panic(&ServeMuxError{err: err, method: method, pattern: pattern})
	}

	if methods[method] != nil {
		panic(duplicateError(method, pattern))
	}

	methods[method] = handler
}

// Get registers the GET handler for the given pattern.
func (mux *ServeMux) Get(pattern string, handler http.Handler) {
	mux.Handle(http.MethodGet, pattern, handler)
}

// Head registers the HEAD handler for the given pattern.
func (mux *ServeMux) Head(pattern string, handler http.Handler) {
	mux.Handle(http.MethodHead, pattern, handler)
}

// Post registers the POST handler for the given pattern.
func (mux *ServeMux) Post(pattern string, handler http.Handler) {
	mux.Handle(http.MethodPost, pattern, handler)
}

// Put registers the PUT handler for the given pattern.
func (mux *ServeMux) Put(pattern string, handler http.Handler) {
	mux.Handle(http.MethodPut, pattern, handler)
}

// Patch registers the PATCH handler for the given pattern.
func (mux *ServeMux) Patch(pattern string, handler http.Handler) {
	mux.Handle(http.MethodPatch, pattern, handler)
}

// Delete registers the DELETE handler for the given pattern.
func (mux *ServeMux) Delete(pattern string, handler http.Handler) {
	mux.Handle(http.MethodDelete, pattern, handler)
}

// Connect registers the CONNECT handler for the given pattern.
func (mux *ServeMux) Connect(pattern string, handler http.Handler) {
	mux.Handle(http.MethodConnect, pattern, handler)
}

// Options registers the OPTIONS handler for the given pattern.
func (mux *ServeMux) Options(pattern string, handler http.Handler) {
	mux.Handle(http.MethodOptions, pattern, handler)
}

// Trace registers the TRACE handler for the given pattern.
func (mux *ServeMux) Trace(pattern string, handler http.Handler) {
	mux.Handle(http.MethodTrace, pattern, handler)
}

// HandleFunc registers the handler function for the given method and pattern.
func (mux *ServeMux) HandleFunc(method, pattern string, handler func(http.ResponseWriter, *http.Request)) {
	if handler == nil {
		panic(handlerError(method, pattern))
	}

	mux.Handle(method, pattern, http.HandlerFunc(handler))
}

// GetFunc registers the GET handler function for the given pattern.
func (mux *ServeMux) GetFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mux.HandleFunc(http.MethodGet, pattern, handler)
}

// HeadFunc registers the HEAD handler function for the given pattern.
func (mux *ServeMux) HeadFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mux.HandleFunc(http.MethodHead, pattern, handler)
}

// PostFunc registers the POST handler function for the given pattern.
func (mux *ServeMux) PostFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mux.HandleFunc(http.MethodPost, pattern, handler)
}

// PutFunc registers the PUT handler function for the given pattern.
func (mux *ServeMux) PutFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mux.HandleFunc(http.MethodPut, pattern, handler)
}

// PatchFunc registers the PATCH handler function for the given pattern.
func (mux *ServeMux) PatchFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mux.HandleFunc(http.MethodPatch, pattern, handler)
}

// DeleteFunc registers the DELETE handler function for the given pattern.
func (mux *ServeMux) DeleteFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mux.HandleFunc(http.MethodDelete, pattern, handler)
}

// ConnectFunc registers the CONNECT handler function for the given pattern.
func (mux *ServeMux) ConnectFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mux.HandleFunc(http.MethodConnect, pattern, handler)
}

// OptionsFunc registers the OPTIONS handler function for the given pattern.
func (mux *ServeMux) OptionsFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mux.HandleFunc(http.MethodOptions, pattern, handler)
}

// TraceFunc registers the TRACE handler function for the given pattern.
func (mux *ServeMux) TraceFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mux.HandleFunc(http.MethodTrace, pattern, handler)
}

// ServeHTTP implements a Handler's interface.
func (mux *ServeMux) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	handler, err := mux.Handler(req)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(writer, req)

		return
	}

	handler.ServeHTTP(writer, req)
}

// New allocates and returns a new ServeMux.
func New() *ServeMux {
	strC := new(convert(strConv))
	intC := new(convert(intConv))

	return &ServeMux{
		tree: &tree{root: &node{
			conv:     nil,
			Methods:  nil,
			Children: nil,
			tid:      root,
		}},
		converters: map[string]*convert{
			"":    strC,
			"str": strC,
			"int": intC,
		},
	}
}
