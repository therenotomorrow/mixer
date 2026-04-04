package mixer

import (
	"net/http"
	"testing"
)

func TestMethodError(t *testing.T) {
	exp := &ServeMuxError{ErrMethod, "method", "pattern"}

	as := Assert{t}
	as.Equal(methodError("method", "pattern"), exp, "methodError() got")
}

func TestHandlerError(t *testing.T) {
	exp := &ServeMuxError{ErrHandler, "method", "pattern"}

	as := Assert{t}
	as.Equal(handlerError("method", "pattern"), exp, "handlerError() got")
}

func TestPatternError(t *testing.T) {
	exp := &ServeMuxError{ErrPattern, "method", "pattern"}

	as := Assert{t}
	as.Equal(patternError("method", "pattern"), exp, "patternError() got")
}

func TestDuplicateError(t *testing.T) {
	exp := &ServeMuxError{ErrDuplicate, "method", "pattern"}

	as := Assert{t}
	as.Equal(duplicateError("method", "pattern"), exp, "duplicateError() got")
}

func TestNotFoundError(t *testing.T) {
	exp := &ServeMuxError{ErrNotFound, "method", "pattern"}

	as := Assert{t}
	as.Equal(notFoundError("method", "pattern"), exp, "notFoundError() got")
}

func TestIntConv(t *testing.T) {
	cases := []struct {
		name string
		s    string
		want int
		err  error
	}{
		{
			name: "valid",
			s:    "123",
			want: 123,
			err:  nil,
		},
		{
			name: "invalid",
			s:    "one-two-three",
			want: 0,
			err:  invalidIntError{"one-two-three"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := intConv(c.s)

			as := Assert{t}
			as.IntEqual(got.(int), c.want, "intConv() got")
			as.Equal(err, c.err, "intConv() error")
		})
	}
}

func TestStrConv(t *testing.T) {
	cases := []struct {
		name string
		s    string
		want string
		err  error
	}{
		{
			name: "valid",
			s:    "one-two-three",
			want: "one-two-three",
			err:  nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := strConv(c.s)

			as := Assert{t}
			as.StrEqual(got.(string), c.want, "strConv() got")
			as.Equal(err, c.err, "strConv() error")
		})
	}
}

func TestSplitURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want []string
		err  error
	}{
		{
			name: "empty url",
			url:  "",
			want: []string{},
			err:  ErrPattern,
		},
		{
			name: "double root",
			url:  "//a/b",
			want: []string{"", "a", "b"},
			err:  ErrPattern,
		},
		{
			name: "double inner slash",
			url:  "/a//b",
			want: []string{"a", "", "b"},
			err:  ErrPattern,
		},
		{
			name: "double trailing slash",
			url:  "/a/b//",
			want: []string{"a", "b", "", "/"},
			err:  ErrPattern,
		},
		{
			name: "without root",
			url:  "a/b",
			want: []string{},
			err:  ErrPattern,
		},
		{
			name: "root only",
			url:  "/",
			want: []string{"/"},
			err:  nil,
		},
		{
			name: "with trailing slash",
			url:  "/a/b/",
			want: []string{"a", "b", "/"},
			err:  nil,
		},
		{
			name: "without trailing slash",
			url:  "/a/b",
			want: []string{"a", "b"},
			err:  nil,
		},
		{
			name: "long url",
			url:  "/a/b/c/d/e/f/",
			want: []string{"a", "b", "c", "d", "e", "f", "/"},
			err:  nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := splitURL(c.url)

			as := Assert{t}
			as.Equal(got, c.want, "splitURL() got")
			as.Equal(err, c.err, "splitURL() error")
		})
	}
}

func TestTreeDeepcopy(t *testing.T) {
	as := Assert{t}

	var (
		ic convert = intConv
		sc convert = strConv
	)

	origin := &tree{
		root: &node{
			conv: &ic,
			Methods: map[string]http.Handler{
				"a": TestHandler("a"),
				"b": TestHandler("b"),
				"c": TestHandler("c"),
			},
			Children: map[string]*node{
				"ch1": {
					conv:     nil,
					Methods:  nil,
					Children: nil,
				},
				"ch2": {
					conv:    &sc,
					tid:     slash,
					Methods: nil,
					Children: map[string]*node{
						"sub-ch1": {
							conv: nil,
							tid:  param,
							Methods: map[string]http.Handler{
								"sub-a": TestHandler("sub-a"),
							},

							Children: map[string]*node{
								"sub-sub-ch1": {
									conv:     nil,
									Methods:  nil,
									Children: nil,
								},
								// "sub-sub-ch2": nil, // this is unexpected situation
							},
						},
					},
				},
			},
		},
	}
	cp := origin.deepcopy()

	as.PtrNotEqual(origin, cp, "tree")

	as.PtrNotEqual(origin.root, cp.root, "tree.root")
	as.PtrEqual(origin.root.conv, cp.root.conv, "tree.root.conv")
	as.IntEqual(origin.root.tid, cp.root.tid, "tree.root.tid")
	as.PtrEqual(origin.root.Methods, cp.root.Methods, "tree.root.Methods")
	as.PtrNotEqual(origin.root.Children, cp.root.Children, "tree.root.Children")

	ch1 := origin.root.Children["ch1"]
	ch1Cp := cp.root.Children["ch1"]

	as.PtrNotEqual(ch1, ch1Cp, "tree.root.Children[ch1]")
	as.PtrEqual(ch1.conv, ch1Cp.conv, "tree.root.Children[ch1].conv")
	as.IntEqual(ch1.tid, ch1Cp.tid, "tree.root.Children[ch1].tid")
	as.PtrEqual(ch1.Methods, ch1Cp.Methods, "tree.root.Children[ch1].Methods")
	// because they are both empty
	as.PtrEqual(ch1.Children, ch1Cp.Children, "tree.root.Children[ch1].Children")

	ch2 := origin.root.Children["ch2"]
	ch2Cp := cp.root.Children["ch2"]

	as.PtrNotEqual(ch2, ch2Cp, "tree.root.Children[ch2]")
	as.PtrEqual(ch2.conv, ch2Cp.conv, "tree.root.Children[ch2].conv")
	as.IntEqual(ch2.tid, ch2Cp.tid, "tree.root.Children[ch2].tid")
	as.PtrEqual(ch2.Methods, ch2Cp.Methods, "tree.root.Children[ch2].Methods")
	as.PtrNotEqual(ch2.Children, ch2Cp.Children, "tree.root.Children[ch2].Children")

	ch2Sub := ch2.Children["sub-ch1"]
	ch2SubCp := ch2Cp.Children["sub-ch1"]

	as.PtrNotEqual(ch2Sub, ch2SubCp, "tree.root.Children[ch2][sub-ch1]")
	as.PtrEqual(ch2Sub.conv, ch2SubCp.conv, "tree.root.Children[ch2][sub-ch1].conv")
	as.IntEqual(ch2Sub.tid, ch2SubCp.tid, "tree.root.Children[ch2][sub-ch1].tid")
	as.PtrEqual(ch2Sub.Methods, ch2SubCp.Methods, "tree.root.Children[ch2][sub-ch1].Methods")
	as.PtrNotEqual(ch2Sub.Children, ch2SubCp.Children, "tree.root.Children[ch2][sub-ch1].Children")

	subSubCh1 := ch2Sub.Children["sub-sub-ch1"]
	subSubCh1Cp := ch2SubCp.Children["sub-sub-ch1"]

	as.PtrNotEqual(subSubCh1, subSubCh1Cp, "tree.root.Children[ch2][sub-ch1][sub-sub-ch1]")
	as.PtrEqual(subSubCh1.conv, subSubCh1Cp.conv, "tree.root.Children[ch2][sub-ch1][sub-sub-ch1].conv")
	as.IntEqual(subSubCh1.tid, subSubCh1Cp.tid, "tree.root.Children[ch2][sub-ch1][sub-sub-ch1].tid")
	as.PtrEqual(subSubCh1.Methods, subSubCh1Cp.Methods, "tree.root.Children[ch2][sub-ch1][sub-sub-ch1].Methods")
	as.PtrEqual(subSubCh1.Children, subSubCh1Cp.Children, "tree.root.Children[ch2][sub-ch1][sub-sub-ch1].Children")
}

func TestNodeFind(t *testing.T) {
	cases := []struct {
		name string
		tid  int
		want *node
	}{
		{
			name: "found",
			tid:  other,
			want: &node{tid: other},
		},
		{
			name: "not found",
			tid:  param,
			want: nil,
		},
	}

	n := &node{
		tid:      slash,
		Children: map[string]*node{"a": {tid: other}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := n.find(c.tid)

			as := Assert{t}
			as.Equal(got, c.want, "node.find() got")
		})
	}
}

func TestNodeInsert(t *testing.T) {
	var conv convert = intConv

	type args struct {
		key string
		in  *node
	}

	cases := []struct {
		name     string
		what     args
		whither  *node
		expected *node
	}{
		{
			name: "param type ID to empty node",
			what: args{"key", &node{
				conv: &conv,
				tid:  param,
			}},
			whither: &node{},
			expected: &node{
				Children: map[string]*node{
					"key": {
						conv: &conv,
						tid:  param,
					},
				},
			},
		},
		{
			name: "other type ID to empty node",
			what: args{"key", &node{
				conv: &conv,
				tid:  other,
			}},
			whither: &node{},
			expected: &node{
				Children: map[string]*node{
					"key": {
						conv: &conv,
						tid:  other,
					},
				},
			},
		},
		{
			name: "param type ID to other node",
			what: args{"key", &node{
				conv: &conv,
				tid:  param,
			}},
			whither: &node{
				Children: map[string]*node{
					"other": {
						tid: other,
					},
				},
			},
			expected: &node{
				Children: map[string]*node{
					"other": {
						tid: other,
					},
					"key": {
						conv: &conv,
						tid:  param,
					},
				},
			},
		},
		{
			name: "other type ID to param node",
			what: args{"key", &node{
				conv: &conv,
				tid:  other,
			}},
			whither: &node{
				Children: map[string]*node{
					"other": {
						conv: &conv,
						tid:  param,
					},
				},
			},
			expected: &node{
				Children: map[string]*node{
					"other": {
						conv: &conv,
						tid:  param,
					},
					"key": {
						conv: &conv,
						tid:  other,
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.whither.insert(c.what.key, c.what.in)

			as := Assert{t}
			as.EqualIndent(c.whither, c.expected, "node.insert() struct")
		})
	}
}

func TestServeMuxAddLogicCases(t *testing.T) {
	mux := New() // for direct compatibility (for not to remap the converters)
	exp := &tree{root: &node{tid: root}}
	as := Assert{t}

	parts := []string{"a", "b"}
	exp.root.Children = map[string]*node{
		"a": {Children: map[string]*node{
			"b": {Methods: map[string]http.Handler{}},
		}},
	}
	_, err := mux.insert(parts)

	as.Equal(err, nil, "without trailing slash")
	as.EqualIndent(mux.tree, exp, "without trailing slash")

	parts = []string{"a", "b", "/"}
	exp.root.
		Children["a"].
		Children["b"].
		Children = map[string]*node{
		"/": {
			tid:     slash,
			Methods: map[string]http.Handler{},
		},
	}
	_, err = mux.insert(parts)

	as.Equal(err, nil, "with trailing slash")
	as.EqualIndent(mux.tree, exp, "with trailing slash")

	parts = []string{"a", "d"}
	exp.root.
		Children["a"].
		Children["d"] = &node{Methods: map[string]http.Handler{}}
	_, err = mux.insert(parts)

	as.Equal(err, nil, "split paths")
	as.EqualIndent(mux.tree, exp, "split paths")

	parts = []string{"a", "b", ":int"}
	exp.root.
		Children["a"].
		Children["b"].
		Children[":"] = &node{tid: param, conv: mux.converters["int"], Methods: map[string]http.Handler{}}
	_, err = mux.insert(parts)

	as.Equal(err, nil, "typed path param")
	as.EqualIndent(mux.tree, exp, "typed path param")

	_, err = mux.insert(parts)
	as.Equal(err, nil, "duplicate typed path param")
	as.EqualIndent(mux.tree, exp, "duplicate typed path param")

	parts = []string{"/"}
	exp.root.
		Children["/"] = &node{tid: slash, Methods: map[string]http.Handler{}}
	hm, err := mux.insert(parts)

	as.Equal(err, nil, "add root")
	as.EqualIndent(mux.tree, exp, "add root")

	hm[http.MethodGet] = TestHandler("/")
	exp.root.Children["/"].Methods[http.MethodGet] = TestHandler("/")
	_, err = mux.insert(parts)

	as.Equal(err, nil, "correct handler map")
	as.EqualIndent(mux.tree, exp, "correct handler map")

	parts = []string{"a", "b", ":int", ":"}
	exp.root.
		Children["a"].
		Children["b"].
		Children[":"].
		Children = map[string]*node{
		":": {
			tid:     param,
			conv:    mux.converters[""],
			Methods: map[string]http.Handler{http.MethodPut: TestHandler("a/b/:int/:")},
		},
	}
	hm, err = mux.insert(parts)
	hm[http.MethodPut] = TestHandler("a/b/:int/:")

	as.Equal(err, nil, "correct handler map and another converter")
	as.EqualIndent(mux.tree, exp, "correct handler map and another converter")

	parts = []string{"a", ":int"}
	exp.root.
		Children["a"].
		Children[":"] = &node{tid: param, conv: mux.converters["int"], Methods: map[string]http.Handler{}}
	_, err = mux.insert(parts)

	as.Equal(err, nil, "different type (b vs. :int)")
	as.EqualIndent(mux.tree, exp, "different type (b vs. :int)")

	parts = []string{"a", "b", ":str"}
	_, err = mux.insert(parts)

	as.Equal(err, ErrMultiplePathParam, "different type (:int vs. :str)")
	as.EqualIndent(mux.tree, exp, "different type (:int vs. :str)")

	parts = []string{"a", "b", "c"}
	exp.root.
		Children["a"].
		Children["b"].
		Children["c"] = &node{Methods: map[string]http.Handler{}}
	_, err = mux.insert(parts)

	as.Equal(err, nil, "different type (:int vs. c)")
	as.EqualIndent(mux.tree, exp, "different type (:int vs. c)")

	parts = []string{"a", "b", ":int", "/"}
	exp.root.
		Children["a"].
		Children["b"].
		Children[":"].
		Children["/"] = &node{tid: slash, Methods: map[string]http.Handler{}}
	_, err = mux.insert(parts)

	as.Equal(err, nil, "different type (:int vs. /)")
	as.EqualIndent(mux.tree, exp, "different type (:int vs. /)")

	parts = []string{"a", "b", ":int", ":str", "c"}
	exp.root.
		Children["a"].
		Children["b"].
		Children[":"].
		Children[":"].
		Children = map[string]*node{
		"c": {
			Methods: map[string]http.Handler{},
		},
	}
	_, err = mux.insert(parts)

	as.Equal(err, nil, "invariant conv")
	as.EqualIndent(mux.tree, exp, "invariant conv")

	parts = []string{"a", "b", ":int", ":", "/"}
	exp.root.
		Children["a"].
		Children["b"].
		Children[":"].
		Children[":"].
		Children["/"] = &node{tid: slash, Methods: map[string]http.Handler{}}
	_, err = mux.insert(parts)

	as.Equal(err, nil, "invariant for /")
	as.EqualIndent(mux.tree, exp, "invariant for /")

	parts = []string{"a", "b", ":int", ":", ":str"}
	exp.root.
		Children["a"].
		Children["b"].
		Children[":"].
		Children[":"].
		Children[":"] = &node{
		tid:     param,
		conv:    mux.converters["str"],
		Methods: map[string]http.Handler{},
	}
	_, err = mux.insert(parts)

	as.Equal(err, nil, "prevent /, c and : together")
	as.EqualIndent(mux.tree, exp, "prevent /, c and : together")

	parts = []string{"a", "b", ":mem"}
	_, err = mux.insert(parts)

	as.Equal(err, ErrPathParam, "invalid path param")
	as.EqualIndent(mux.tree, exp, "invalid path param")

	parts = []string{"g", "g", "w", "p", ":gl"}
	_, err = mux.insert(parts)

	as.Equal(err, ErrPathParam, "deep copy valid (new path)")
	as.EqualIndent(mux.tree, exp, "deep copy valid (new path)")

	parts = []string{"a", "b", ":int", ":", "a", "b", ":hf"}
	_, err = mux.insert(parts)

	as.Equal(err, ErrPathParam, "deep copy valid (exist path)")
	as.EqualIndent(mux.tree, exp, "deep copy valid (exist path)")
}

func TestServeMuxAddDirectCases(t *testing.T) {
	mux := New() // for direct compatibility (for not to remap the converters)

	cases := []struct {
		name     string
		parts    []string
		root     *node
		wantRoot *node
		want     error
	}{
		{
			name:  "a vs. :int",
			parts: []string{":int"},
			root: &node{
				Children: map[string]*node{
					"a": {Methods: map[string]http.Handler{}},
				},
			},
			wantRoot: &node{
				Children: map[string]*node{
					"a": {Methods: map[string]http.Handler{}},
					":": {tid: param, conv: mux.converters["int"], Methods: map[string]http.Handler{}},
				},
			},
			want: nil,
		},
		{
			name:  "a vs. /",
			parts: []string{"/"},
			root: &node{
				Children: map[string]*node{
					"a": {Methods: map[string]http.Handler{}},
				},
			},
			wantRoot: &node{
				Children: map[string]*node{
					"a": {Methods: map[string]http.Handler{}},
					"/": {tid: slash, Methods: map[string]http.Handler{}},
				},
			},
			want: nil,
		},
		{
			name:  ":int vs. a",
			parts: []string{"a"},
			root: &node{
				Children: map[string]*node{
					":": {tid: param, conv: mux.converters["int"], Methods: map[string]http.Handler{}},
				},
			},
			wantRoot: &node{
				Children: map[string]*node{
					":": {tid: param, conv: mux.converters["int"], Methods: map[string]http.Handler{}},
					"a": {Methods: map[string]http.Handler{}},
				},
			},
			want: nil,
		},
		{
			name:  ":int vs. /",
			parts: []string{"/"},
			root: &node{
				Children: map[string]*node{
					":": {tid: param, conv: mux.converters["int"], Methods: map[string]http.Handler{}},
				},
			},
			wantRoot: &node{
				Children: map[string]*node{
					":": {tid: param, conv: mux.converters["int"], Methods: map[string]http.Handler{}},
					"/": {tid: slash, Methods: map[string]http.Handler{}},
				},
			},
			want: nil,
		},
		{
			name:  "/ vs. a",
			parts: []string{"a"},
			root: &node{
				Children: map[string]*node{
					"/": {tid: slash, Methods: map[string]http.Handler{}},
				},
			},
			wantRoot: &node{
				Children: map[string]*node{
					"/": {tid: slash, Methods: map[string]http.Handler{}},
					"a": {Methods: map[string]http.Handler{}},
				},
			},
			want: nil,
		},
		{
			name:  "/ vs. :str",
			parts: []string{":str"},
			root: &node{
				Children: map[string]*node{
					"/": {tid: slash, Methods: map[string]http.Handler{}},
				},
			},
			wantRoot: &node{
				Children: map[string]*node{
					"/": {tid: slash, Methods: map[string]http.Handler{}},
					":": {tid: param, conv: mux.converters["str"], Methods: map[string]http.Handler{}},
				},
			},
			want: nil,
		},
		{
			name:  "a vs. b",
			parts: []string{"b"},
			root: &node{
				Children: map[string]*node{
					"a": {Methods: map[string]http.Handler{}},
				},
			},
			wantRoot: &node{
				Children: map[string]*node{
					"a": {Methods: map[string]http.Handler{}},
					"b": {Methods: map[string]http.Handler{}},
				},
			},
			want: nil,
		},
		{
			name:  ":int vs. :",
			parts: []string{":"},
			root: &node{
				Children: map[string]*node{
					":": {tid: param, conv: mux.converters["int"], Methods: map[string]http.Handler{}},
				},
			},
			wantRoot: &node{
				Children: map[string]*node{
					":": {tid: param, conv: mux.converters["int"], Methods: map[string]http.Handler{}},
				},
			},
			want: ErrMultiplePathParam,
		},
		{
			name:  ": vs. :mem",
			parts: []string{":mem"},
			root: &node{
				Children: map[string]*node{
					":": {tid: param, conv: mux.converters[""], Methods: map[string]http.Handler{}},
				},
			},
			wantRoot: &node{
				Children: map[string]*node{
					":": {tid: param, conv: mux.converters[""], Methods: map[string]http.Handler{}},
				},
			},
			want: ErrPathParam,
		},
		{
			name:  ": vs. :str",
			parts: []string{":str"},
			root: &node{
				Children: map[string]*node{
					":": {tid: param, conv: mux.converters[""], Methods: map[string]http.Handler{}},
				},
			},
			wantRoot: &node{
				Children: map[string]*node{
					":": {tid: param, conv: mux.converters["str"], Methods: map[string]http.Handler{}},
				},
			},
			want: nil,
		},
		{
			name:  "/ and :int vs. a",
			parts: []string{"a"},
			root: &node{
				Children: map[string]*node{
					"/": {tid: slash, Methods: map[string]http.Handler{}},
					":": {tid: param, conv: mux.converters["int"], Methods: map[string]http.Handler{}},
				},
			},
			wantRoot: &node{
				Children: map[string]*node{
					"/": {tid: slash, Methods: map[string]http.Handler{}},
					":": {tid: param, conv: mux.converters["int"], Methods: map[string]http.Handler{}},
					"a": {Methods: map[string]http.Handler{}},
				},
			},
			want: nil,
		},
		{
			name:  "/ and a vs. :int",
			parts: []string{":int"},
			root: &node{
				Children: map[string]*node{
					"/": {tid: slash, Methods: map[string]http.Handler{}},
					"a": {Methods: map[string]http.Handler{}},
				},
			},
			wantRoot: &node{
				Children: map[string]*node{
					"/": {tid: slash, Methods: map[string]http.Handler{}},
					"a": {Methods: map[string]http.Handler{}},
					":": {tid: param, conv: mux.converters["int"], Methods: map[string]http.Handler{}},
				},
			},
			want: nil,
		},
		{
			name:  "add sub node",
			parts: []string{"b", "c"},
			root: &node{
				Children: map[string]*node{
					"/": {tid: slash, Methods: map[string]http.Handler{}},
				},
			},
			wantRoot: &node{
				Children: map[string]*node{
					"/": {tid: slash, Methods: map[string]http.Handler{}},
					"b": {
						Children: map[string]*node{
							"c": {Methods: map[string]http.Handler{}},
						},
					},
				},
			},
			want: nil,
		},
		{
			name:  "prevent create nodes if error was deeper",
			parts: []string{"a", "b", ":mem"},
			root: &node{
				Children: map[string]*node{
					"/": {tid: slash, Methods: map[string]http.Handler{}},
				},
			},
			wantRoot: &node{
				Children: map[string]*node{
					"/": {tid: slash, Methods: map[string]http.Handler{}},
				},
			},
			want: ErrPathParam,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mux.tree.root = c.root

			_, got := mux.insert(c.parts)

			as := Assert{t}
			as.Equal(got, c.want, "ServeMux.add() error")
			as.EqualIndent(mux.tree.root, c.wantRoot, "ServeMux.add() tree")

			mux.tree.root = nil
		})
	}
}

func TestInvalidIntErrorError(t *testing.T) {
	err := invalidIntError{val: "hello world"}

	as := Assert{t}
	as.StrEqual(`mixer: invalid int "hello world"`, err.Error(), "invalidIntError.Error")
}
