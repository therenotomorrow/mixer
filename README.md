mixer
=====

`mixer` is a simple parametrized router for building HTTP services.

The designed API is similar to [httprouter.Router](https://godoc.org/github.com/julienschmidt/httprouter#Router)
but allows us to use the typed path params.

Install
-------

```bash
$ go get -u github.com/therenotomorrow/mixer
```

How it works
------------

The `ServeMux` based on the similar [Radix Tree](https://en.wikipedia.org/wiki/Radix_tree) data structure.
Let's see at URL examples:

```
/
/catalog/
/catalog/:int
/catalog/:int/items/
/catalog/:int/items/:int
```

The inner tree structure will be like this:

![tree](tree.png)

And you can see that URL will be separated by parts and search will be down to leaf where the URL is registered.
And the nodes contain only one part or special part like `:` for typed param and `/` for trailing slash.

Important behavior
------------------

- `ServeMux` is not safe for concurrent route registration.
- Static segments and typed params can coexist on the same level, both `/users/me` and `/users/:str` are valid.
- Route matching prefers the static segment first, then typed param fallback.
- `Handle` and `HandleFunc` panic on invalid setup.

What about API?
---------------

All available API described in [api.go](api.go). Here is the full list:

```go
// GetPathParams returns the path params registered in r.Context() or nil otherwise.
func GetPathParams(r *http.Request) PathParams {}

// Handler returns the handler to use for the given request.
func (mux *ServeMux) Handler(r *http.Request) (http.Handler, error)

// Handle registers the handler for the given method and pattern.
func (mux *ServeMux) Handle(method, pattern string, handler http.Handler) 

// Get registers the GET handler for the given pattern.
func (mux *ServeMux) Get(pattern string, handler http.Handler)

// Head registers the HEAD handler for the given pattern.
func (mux *ServeMux) Head(pattern string, handler http.Handler)

// Post registers the POST handler for the given pattern.
func (mux *ServeMux) Post(pattern string, handler http.Handler)

// Put registers the PUT handler for the given pattern.
func (mux *ServeMux) Put(pattern string, handler http.Handler)

// Patch registers the PATCH handler for the given pattern.
func (mux *ServeMux) Patch(pattern string, handler http.Handler)

// Delete registers the DELETE handler for the given pattern.
func (mux *ServeMux) Delete(pattern string, handler http.Handler)

// Connect registers the CONNECT handler for the given pattern.
func (mux *ServeMux) Connect(pattern string, handler http.Handler)

// Options registers the OPTIONS handler for the given pattern.
func (mux *ServeMux) Options(pattern string, handler http.Handler)

// Trace registers the TRACE handler for the given pattern.
func (mux *ServeMux) Trace(pattern string, handler http.Handler)

// HandleFunc registers the handler function for the given method and pattern.
func (mux *ServeMux) HandleFunc(method, pattern string, handler func(http.ResponseWriter, *http.Request))

// GetFunc registers the GET handler function for the given pattern.
func (mux *ServeMux) GetFunc(pattern string, handler func(http.ResponseWriter, *http.Request))

// HeadFunc registers the HEAD handler function for the given pattern.
func (mux *ServeMux) HeadFunc(pattern string, handler func(http.ResponseWriter, *http.Request))

// PostFunc registers the POST handler function for the given pattern.
func (mux *ServeMux) PostFunc(pattern string, handler func(http.ResponseWriter, *http.Request))

// PutFunc registers the PUT handler function for the given pattern.
func (mux *ServeMux) PutFunc(pattern string, handler func(http.ResponseWriter, *http.Request))

// PatchFunc registers the PATCH handler function for the given pattern.
func (mux *ServeMux) PatchFunc(pattern string, handler func(http.ResponseWriter, *http.Request))

// DeleteFunc registers the DELETE handler function for the given pattern.
func (mux *ServeMux) DeleteFunc(pattern string, handler func(http.ResponseWriter, *http.Request))

// ConnectFunc registers the CONNECT handler function for the given pattern.
func (mux *ServeMux) ConnectFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) 

// OptionsFunc registers the OPTIONS handler function for the given pattern.
func (mux *ServeMux) OptionsFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) 

// TraceFunc registers the TRACE handler function for the given pattern.
func (mux *ServeMux) TraceFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) 

// ServeHTTP implements the http.Handler interface.
func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request)

// New allocates and returns a new ServeMux.
func New() *ServeMux
```
