package mixer

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeMuxErrorError(t *testing.T) {
	err := ServeMuxError{
		method:  "method",
		pattern: "pattern",
		err:     errors.New("error"),
	}
	want := "mixer: handler (method) pattern error: error"

	as := Assert{t}
	as.StrEqual(err.Error(), want, "ServeMuxError.Error() got")
}

func TestServeMuxErrorUnwrap(t *testing.T) {
	err := ServeMuxError{
		method:  "method",
		pattern: "pattern",
		err:     errors.New("error"),
	}
	want := errors.New("error")

	as := Assert{t}
	as.Equal(err.Unwrap(), want, "ServeMuxError.Unwrap() got")
}

func TestGetPathParams(t *testing.T) {
	var exp PathParams // empty

	ctx := context.Background()
	req := mustReq(http.NewRequestWithContext(ctx, "", "", http.NoBody))

	as := Assert{t}
	as.Equal(GetPathParams(req), exp, "params not set")

	ctx = context.WithValue(ctx, contextKey("wrong-key"), PathParams{0: 12, 1: "abc"})
	req = mustReq(http.NewRequestWithContext(ctx, "", "", http.NoBody))

	as.Equal(GetPathParams(req), exp, "wrong context key")

	ctx = context.WithValue(ctx, PathParamsCtxKey, PathParams{0: 12, 1: "abc"})
	req = mustReq(http.NewRequestWithContext(ctx, "", "", http.NoBody))
	exp = PathParams{0: 12, 1: "abc"}

	as.Equal(GetPathParams(req), exp, "path params exist")
}

func TestServeMuxHandlerLogicCases(t *testing.T) {
	mux := New() // for direct compatibility (for not to remap the converters)
	mux.tree.root = &node{Children: map[string]*node{
		"/": {
			tid: root,
			Methods: map[string]http.Handler{
				http.MethodGet: TestHandler("get"),
			},
		},
		"a": {
			Methods: map[string]http.Handler{
				http.MethodPut: TestHandler("put"),
			},
			Children: map[string]*node{
				":": {
					tid:  param,
					conv: mux.converters["int"],
					Methods: map[string]http.Handler{
						http.MethodPost:  TestHandler("post"),
						http.MethodPatch: TestHandler("patch"),
					},
					Children: map[string]*node{
						"/": {
							tid: slash,
							Methods: map[string]http.Handler{
								http.MethodGet: TestHandler("trailing slash"),
							},
						},
						"b": {
							Children: map[string]*node{
								":": {
									tid:  param,
									conv: mux.converters["str"],
									Methods: map[string]http.Handler{
										http.MethodDelete: TestHandler("delete"),
									},
								},
							},
						},
						"a": {
							Children: map[string]*node{
								"c": {
									Methods: map[string]http.Handler{
										http.MethodGet: TestHandler("last"),
									},
								},
							},
						},
					},
				},
			},
		},
	}}

	req := mustReq(http.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody))
	ctx := context.Background()
	got, err := mux.Handler(req)

	as := Assert{t}
	as.Equal(got, TestHandler("get"), "for / got")
	as.Equal(err, nil, "for / error")
	as.Equal(req.Context(), ctx, "for / context")

	req = mustReq(http.NewRequestWithContext(context.Background(), http.MethodPut, "/", http.NoBody))
	got, err = mux.Handler(req)

	as.Equal(got, nil, "non-exist handler got")
	as.Equal(err, notFoundError(http.MethodPut, "/"), "non-exist handler error")
	as.Equal(req.Context(), ctx, "non-exist handler context")

	req = mustReq(http.NewRequestWithContext(context.Background(), http.MethodPut, "/a", http.NoBody))
	got, err = mux.Handler(req)

	as.Equal(got, TestHandler("put"), "for /a got")
	as.Equal(err, nil, "for /a error")
	as.Equal(req.Context(), ctx, "for /a context")

	req = mustReq(http.NewRequestWithContext(context.Background(), http.MethodPost, "/a/123", http.NoBody))
	ctx = context.WithValue(ctx, PathParamsCtxKey, PathParams{0: 123})
	got, err = mux.Handler(req)

	as.Equal(got, TestHandler("post"), "for int got")
	as.Equal(err, nil, "for int error")
	as.Equal(req.Context(), ctx, "for int context")

	req = mustReq(http.NewRequestWithContext(context.Background(), http.MethodPost, "/a/one_two_three", http.NoBody))
	ctx = context.Background()
	got, err = mux.Handler(req)

	as.Equal(got, nil, "for int wrong type got")
	as.Equal(err, notFoundError(http.MethodPost, "/a/one_two_three"), "for int wrong type error")
	as.Equal(req.Context(), ctx, "for int wrong type context")

	ctx = context.WithValue(context.Background(), contextKey("old-context"), "oldContext")
	req = mustReq(http.NewRequestWithContext(ctx, http.MethodPost, "/a/321", http.NoBody))
	ctx = context.WithValue(ctx, PathParamsCtxKey, PathParams{0: 321})
	got, err = mux.Handler(req)

	as.Equal(got, TestHandler("post"), "save context for original request got")
	as.Equal(err, nil, "save context for original request error")
	as.Equal(req.Context(), ctx, "save context for original request context")

	req = mustReq(http.NewRequestWithContext(context.Background(), http.MethodPost, "/123", http.NoBody))
	ctx = context.Background()
	got, err = mux.Handler(req)

	as.Equal(got, nil, "no handler for param got")
	as.Equal(err, notFoundError(http.MethodPost, "/123"), "no handler for param error")
	as.Equal(req.Context(), ctx, "no handler for param context")

	req = mustReq(http.NewRequestWithContext(context.Background(), http.MethodDelete, "/a/12/b/abc", http.NoBody))
	ctx = context.WithValue(ctx, PathParamsCtxKey, PathParams{0: 12, 1: "abc"})
	got, err = mux.Handler(req)

	as.Equal(got, TestHandler("delete"), "multiple path params got")
	as.Equal(err, nil, "multiple path params error")
	as.Equal(req.Context(), ctx, "multiple path params context")

	mux = New() // create new for clean up the tree
	got, err = mux.Handler(req)

	as.Equal(got, nil, "fresh ServeMux got")
	as.Equal(err, notFoundError(http.MethodDelete, "/a/12/b/abc"), "fresh ServeMux error")
	as.Equal(req.Context(), ctx, "fresh ServeMux context")
}

func TestServeMuxHandlerPreferStaticOverParam(t *testing.T) {
	mux := New()
	mux.Get("/catalog/me", TestHandler("static"))
	mux.Get("/catalog/:str", TestHandler("param"))

	req := mustReq(http.NewRequestWithContext(context.Background(), http.MethodGet, "/catalog/me", http.NoBody))
	got, err := mux.Handler(req)

	as := Assert{t}
	as.Equal(got, TestHandler("static"), "prefer static route")
	as.Equal(err, nil, "prefer static route error")
	as.Equal(GetPathParams(req), PathParams(nil), "prefer static route params")

	req = mustReq(http.NewRequestWithContext(context.Background(), http.MethodGet, "/catalog/abc", http.NoBody))
	got, err = mux.Handler(req)

	as.Equal(got, TestHandler("param"), "fallback to param route")
	as.Equal(err, nil, "fallback to param route error")
	as.Equal(GetPathParams(req), PathParams{0: "abc"}, "fallback to param route params")
}

func TestServeMuxHandlerNotFoundWhenMethodsNil(t *testing.T) {
	mux := New()
	mux.tree.root = &node{
		Children: map[string]*node{
			"catalog": {},
		},
	}

	req := mustReq(http.NewRequestWithContext(context.Background(), http.MethodGet, "/catalog", http.NoBody))
	got, err := mux.Handler(req)

	as := Assert{t}
	as.Equal(got, nil, "methods nil handler")
	as.Equal(err, notFoundError(http.MethodGet, "/catalog"), "methods nil error")
}

func TestServeMuxHandle(t *testing.T) {
	mux := New() // for direct compatibility (for not allocate tree)
	exp := New() // for direct compatibility (for not allocate tree)

	t.Run("panic if invalid method", func(t *testing.T) {
		as := Assert{t}

		defer func() {
			err := recover()
			if err == nil || !errors.Is(errors.Unwrap(err.(error)), ErrMethod) {
				t.Errorf("ServeMux.Handle() got = %v, want = %v", err, ErrMethod)
			}

			as.EqualIndent(mux.tree, exp.tree, "ServeMux.Handle() tree")
		}()

		mux.Handle("invalid method", "/a/b/c/", TestHandler("handler"))
	})

	t.Run("panic if nil handler", func(t *testing.T) {
		as := Assert{t}

		defer func() {
			err := recover()
			if err == nil || !errors.Is(errors.Unwrap(err.(error)), ErrHandler) {
				t.Errorf("ServeMux.Handle() got = %v, want = %v", err, ErrHandler)
			}

			as.EqualIndent(mux.tree, exp.tree, "ServeMux.Handle() tree")
		}()

		mux.Handle(http.MethodGet, "/a/b/c/", nil)
	})

	t.Run("panic if invalid pattern", func(t *testing.T) {
		as := Assert{t}

		defer func() {
			err := recover()
			if err == nil || !errors.Is(errors.Unwrap(err.(error)), ErrPattern) {
				t.Errorf("ServeMux.Handle() got = %v, want = %v", err, ErrPattern)
			}

			as.EqualIndent(mux.tree, exp.tree, "ServeMux.Handle() tree")
		}()

		mux.Handle(http.MethodGet, "/a/b//c/", TestHandler("handler"))
	})

	t.Run("panic if cannot add to tree", func(t *testing.T) {
		as := Assert{t}

		defer func() {
			err := recover()
			if err == nil || errors.Unwrap(err.(error)).Error() != "invalid path param" {
				t.Errorf("ServeMux.Handle() got = %v, want = %v", err, "invalid path param")
			}

			as.EqualIndent(mux.tree, exp.tree, "ServeMux.Handle() tree")
		}()

		mux.Handle(http.MethodGet, "/a/:mem/", TestHandler("handler"))
	})

	mux.tree.root = &node{Children: map[string]*node{
		"/": {tid: slash, Methods: map[string]http.Handler{http.MethodGet: TestHandler("handler")}},
	}}
	exp.tree.root = &node{Children: map[string]*node{
		"/": {tid: slash, Methods: map[string]http.Handler{http.MethodGet: TestHandler("handler")}},
	}}

	t.Run("panic on duplicate handler", func(t *testing.T) {
		as := Assert{t}

		defer func() {
			err := recover()
			if err == nil || !errors.Is(errors.Unwrap(err.(error)), ErrDuplicate) {
				t.Errorf("ServeMux.Handle() got = %v, want = %v", err, ErrDuplicate)
			}

			as.EqualIndent(mux.tree, exp.tree, "ServeMux.Handle() tree")
		}()

		mux.Handle(http.MethodGet, "/", TestHandler("another handler"))
	})

	exp.tree.root.Children["/"].Methods[http.MethodPut] = TestHandler("another handler")

	t.Run("success add handler", func(t *testing.T) {
		mux.Handle(http.MethodPut, "/", TestHandler("another handler"))

		as := Assert{t}
		as.EqualIndent(mux.tree, exp.tree, "ServeMux.Handle() tree")
	})
}

func TestServeMuxGet(t *testing.T) {
	mux := New() // for direct compatibility (for not allocate tree)
	exp := &node{
		tid: root,
		Children: map[string]*node{
			"/": {
				tid: slash,
				Methods: map[string]http.Handler{
					http.MethodGet: TestHandler("get"),
				},
			},
		},
	}

	mux.Get("/", TestHandler("get"))

	as := Assert{t}
	as.Equal(mux.tree.root, exp, "ServeMux.Get() tree")
}

func TestServeMuxHead(t *testing.T) {
	mux := New() // for direct compatibility (for not allocate tree)
	exp := &node{
		tid: root,
		Children: map[string]*node{
			"/": {
				tid: slash,
				Methods: map[string]http.Handler{
					http.MethodHead: TestHandler("head"),
				},
			},
		},
	}

	mux.Head("/", TestHandler("head"))

	as := Assert{t}
	as.Equal(mux.tree.root, exp, "ServeMux.Head() tree")
}

func TestServeMuxPost(t *testing.T) {
	mux := New() // for direct compatibility (for not allocate tree)
	exp := &node{
		tid: root,
		Children: map[string]*node{
			"/": {
				tid: slash,
				Methods: map[string]http.Handler{
					http.MethodPost: TestHandler("post"),
				},
			},
		},
	}

	mux.Post("/", TestHandler("post"))

	as := Assert{t}
	as.Equal(mux.tree.root, exp, "ServeMux.Post() tree")
}

func TestServeMuxPut(t *testing.T) {
	mux := New() // for direct compatibility (for not allocate tree)
	exp := &node{
		tid: root,
		Children: map[string]*node{
			"/": {
				tid: slash,
				Methods: map[string]http.Handler{
					http.MethodPut: TestHandler("put"),
				},
			},
		},
	}

	mux.Put("/", TestHandler("put"))

	as := Assert{t}
	as.Equal(mux.tree.root, exp, "ServeMux.Put() tree")
}

func TestServeMuxPatch(t *testing.T) {
	mux := New() // for direct compatibility (for not allocate tree)
	exp := &node{
		tid: root,
		Children: map[string]*node{
			"/": {
				tid: slash,
				Methods: map[string]http.Handler{
					http.MethodPatch: TestHandler("patch"),
				},
			},
		},
	}

	mux.Patch("/", TestHandler("patch"))

	as := Assert{t}
	as.Equal(mux.tree.root, exp, "ServeMux.Patch() tree")
}

func TestServeMuxDelete(t *testing.T) {
	mux := New() // for direct compatibility (for not allocate tree)
	exp := &node{
		tid: root,
		Children: map[string]*node{
			"/": {
				tid: slash,
				Methods: map[string]http.Handler{
					http.MethodDelete: TestHandler("delete"),
				},
			},
		},
	}

	mux.Delete("/", TestHandler("delete"))

	as := Assert{t}
	as.Equal(mux.tree.root, exp, "ServeMux.Delete() tree")
}

func TestServeMuxConnect(t *testing.T) {
	mux := New() // for direct compatibility (for not allocate tree)
	exp := &node{
		tid: root,
		Children: map[string]*node{
			"/": {
				tid: slash,
				Methods: map[string]http.Handler{
					http.MethodConnect: TestHandler("connect"),
				},
			},
		},
	}

	mux.Connect("/", TestHandler("connect"))

	as := Assert{t}
	as.Equal(mux.tree.root, exp, "ServeMux.Connect() tree")
}

func TestServeMuxOptions(t *testing.T) {
	mux := New() // for direct compatibility (for not allocate tree)
	exp := &node{
		tid: root,
		Children: map[string]*node{
			"/": {
				tid: slash,
				Methods: map[string]http.Handler{
					http.MethodOptions: TestHandler("options"),
				},
			},
		},
	}

	mux.Options("/", TestHandler("options"))

	as := Assert{t}
	as.Equal(mux.tree.root, exp, "ServeMux.Options() tree")
}

func TestServeMuxTrace(t *testing.T) {
	mux := New() // for direct compatibility (for not allocate tree)
	exp := &node{
		tid: root,
		Children: map[string]*node{
			"/": {
				tid: slash,
				Methods: map[string]http.Handler{
					http.MethodTrace: TestHandler("trace"),
				},
			},
		},
	}

	mux.Trace("/", TestHandler("trace"))

	as := Assert{t}
	as.Equal(mux.tree.root, exp, "ServeMux.Trace() tree")
}

func TestServeMuxHandleFunc(t *testing.T) {
	mux := New() // for direct compatibility (for not allocate tree)
	exp := New() // for direct compatibility (for not allocate tree)

	t.Run("panic if nil handler", func(t *testing.T) {
		as := Assert{t}

		defer func() {
			err := recover()
			if err == nil || !errors.Is(errors.Unwrap(err.(error)), ErrHandler) {
				t.Errorf("ServeMux.HandleFunc() got = %v, want = %v", err, ErrHandler)
			}

			as.EqualIndent(mux.tree, exp.tree, "ServeMux.HandleFunc() tree")
		}()

		mux.HandleFunc(http.MethodGet, "/a/b/c/", nil)
	})

	t.Run("success add handler", func(t *testing.T) {
		mux.HandleFunc(http.MethodPut, "/", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok"))
		})

		if mux.tree.root.Children["/"].Methods[http.MethodPut] == nil {
			t.Errorf("ServeMux.HandleFunc() does not add handler")
		}
	})
}

func TestServeMuxGetFunc(t *testing.T) {
	fn := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

	mux := New() // for direct compatibility (for not allocate tree)
	resp := httptest.ResponseRecorder{}

	mux.GetFunc("/", fn)

	// cannot compare handlers because functions can only compare with nil
	mux.tree.root.Children["/"].Methods[http.MethodGet].ServeHTTP(&resp, nil)

	as := Assert{t}
	as.IntEqual(resp.Code, http.StatusOK, "ServeMux.GetFunc()")
}

func TestServeMuxHeadFunc(t *testing.T) {
	fn := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) }

	mux := New() // for direct compatibility (for not allocate tree)
	resp := httptest.ResponseRecorder{}

	mux.HeadFunc("/", fn)

	// cannot compare handlers because functions can only compare with nil
	mux.tree.root.Children["/"].Methods[http.MethodHead].ServeHTTP(&resp, nil)

	as := Assert{t}
	as.IntEqual(resp.Code, http.StatusCreated, "ServeMux.HeadFunc()")
}

func TestServeMuxPostFunc(t *testing.T) {
	fn := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) }

	mux := New() // for direct compatibility (for not allocate tree)
	resp := httptest.ResponseRecorder{}

	mux.PostFunc("/", fn)

	// cannot compare handlers because functions can only compare with nil
	mux.tree.root.Children["/"].Methods[http.MethodPost].ServeHTTP(&resp, nil)

	as := Assert{t}
	as.IntEqual(resp.Code, http.StatusAccepted, "ServeMux.PostFunc()")
}

func TestServeMuxPutFunc(t *testing.T) {
	fn := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNonAuthoritativeInfo) }

	mux := New() // for direct compatibility (for not allocate tree)
	resp := httptest.ResponseRecorder{}

	mux.PutFunc("/", fn)

	// cannot compare handlers because functions can only compare with nil
	mux.tree.root.Children["/"].Methods[http.MethodPut].ServeHTTP(&resp, nil)

	as := Assert{t}
	as.IntEqual(resp.Code, http.StatusNonAuthoritativeInfo, "ServeMux.PutFunc()")
}

func TestServeMuxPatchFunc(t *testing.T) {
	fn := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }

	mux := New() // for direct compatibility (for not allocate tree)
	resp := httptest.ResponseRecorder{}

	mux.PatchFunc("/", fn)

	// cannot compare handlers because functions can only compare with nil
	mux.tree.root.Children["/"].Methods[http.MethodPatch].ServeHTTP(&resp, nil)

	as := Assert{t}
	as.IntEqual(resp.Code, http.StatusNoContent, "ServeMux.PatchFunc()")
}

func TestServeMuxDeleteFunc(t *testing.T) {
	fn := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusResetContent) }

	mux := New() // for direct compatibility (for not allocate tree)
	resp := httptest.ResponseRecorder{}

	mux.DeleteFunc("/", fn)

	// cannot compare handlers because functions can only compare with nil
	mux.tree.root.Children["/"].Methods[http.MethodDelete].ServeHTTP(&resp, nil)

	as := Assert{t}
	as.IntEqual(resp.Code, http.StatusResetContent, "ServeMux.DeleteFunc()")
}

func TestServeMuxConnectFunc(t *testing.T) {
	fn := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusPartialContent) }

	mux := New() // for direct compatibility (for not allocate tree)
	resp := httptest.ResponseRecorder{}

	mux.ConnectFunc("/", fn)

	// cannot compare handlers because functions can only compare with nil
	mux.tree.root.Children["/"].Methods[http.MethodConnect].ServeHTTP(&resp, nil)

	as := Assert{t}
	as.IntEqual(resp.Code, http.StatusPartialContent, "ServeMux.ConnectFunc()")
}

func TestServeMuxOptionsFunc(t *testing.T) {
	fn := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusMultiStatus) }

	mux := New() // for direct compatibility (for not allocate tree)
	resp := httptest.ResponseRecorder{}

	mux.OptionsFunc("/", fn)

	// cannot compare handlers because functions can only compare with nil
	mux.tree.root.Children["/"].Methods[http.MethodOptions].ServeHTTP(&resp, nil)

	as := Assert{t}
	as.IntEqual(resp.Code, http.StatusMultiStatus, "ServeMux.OptionsFunc()")
}

func TestServeMuxTraceFunc(t *testing.T) {
	fn := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAlreadyReported) }

	mux := New() // for direct compatibility (for not allocate tree)
	resp := httptest.ResponseRecorder{}

	mux.TraceFunc("/", fn)

	// cannot compare handlers because functions can only compare with nil
	mux.tree.root.Children["/"].Methods[http.MethodTrace].ServeHTTP(&resp, nil)

	as := Assert{t}
	as.IntEqual(resp.Code, http.StatusAlreadyReported, "ServeMux.TraceFunc()")
}

func TestServeMuxServeHTTP(t *testing.T) {
	mux := New()
	mux.HandleFunc(http.MethodGet, "/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	tc := ts.Client()
	as := Assert{t}

	respGood, _ := tc.Get(ts.URL)

	defer func() { _ = respGood.Body.Close() }()

	as.IntEqual(respGood.StatusCode, http.StatusOK, "success")

	respBad, _ := tc.Head(ts.URL)

	defer func() { _ = respBad.Body.Close() }()

	as.IntEqual(respBad.StatusCode, http.StatusNotFound, "failure")
}

func TestNew(t *testing.T) {
	mux := New()
	exp := &tree{root: &node{tid: root}}
	ass := Assert{t}

	ass.Equal(mux.tree, exp, "newServeMux() tree")

	for _, name := range []string{"", "str"} {
		conv, ok := mux.converters[name]
		if !ok {
			t.Errorf("New() converter %q not exist", name)
		}

		val, err := (*conv)("abc")
		if err != nil {
			t.Errorf("New() converter %q unexpected error %v", name, err)
		}

		_, ok = val.(string)
		if !ok {
			t.Errorf("New() converter %q wrong return type", name)
		}
	}

	name := "int"

	conv, ok := mux.converters[name]
	if !ok {
		t.Errorf("New() converter %q not exist", name)
	}

	val, err := (*conv)("123")
	if err != nil {
		t.Errorf("New() converter %q unexpected error %v", name, err)
	}

	_, ok = val.(int)
	if !ok {
		t.Errorf("New() converter %q wrong return type", name)
	}
}
