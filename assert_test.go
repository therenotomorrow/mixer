package mixer

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

type Assert struct {
	*testing.T
}

func (as *Assert) IntEqual(n1, n2 int, msg string) {
	if as.integer(n1, n2, true) {
		as.Errorf("%s: integers are not equal (%d != %d)", msg, n1, n2)
	}
}

func (as *Assert) IntNotEqual(n1, n2 int, msg string) {
	if as.integer(n1, n2, false) {
		as.Errorf("%s: integers are equal (%d == %d)", msg, n1, n2)
	}
}

func (as *Assert) StrEqual(n1, n2, msg string) {
	if as.str(n1, n2, true) {
		as.Errorf("%s: strings are not equal (%s != %s)", msg, n1, n2)
	}
}

func (as *Assert) StrNotEqual(n1, n2, msg string) {
	if as.str(n1, n2, false) {
		as.Errorf("%s: strings are equal (%s == %s)", msg, n1, n2)
	}
}

func (as *Assert) BoolEqual(n1, n2 bool, msg string) {
	if as.boolean(n1, n2, true) {
		as.Errorf("%s: boleans are not equal (%t != %t)", msg, n1, n2)
	}
}

func (as *Assert) BoolNotEqual(n1, n2 bool, msg string) {
	if as.boolean(n1, n2, false) {
		as.Errorf("%s: boleans are equal (%t == %t)", msg, n1, n2)
	}
}

func (as *Assert) PtrEqual(n1, n2 any, msg string) {
	if as.ptr(n1, n2, true) {
		as.Errorf("%s: pointers are not equal (%p != %p)", msg, n1, n2)
	}
}

func (as *Assert) PtrNotEqual(n1, n2 any, msg string) {
	if as.ptr(n1, n2, false) {
		as.Errorf("%s: pointers are equal (%p == %p)", msg, n1, n2)
	}
}

func (as *Assert) Equal(n1, n2 any, msg string) {
	if !reflect.DeepEqual(n1, n2) {
		as.Errorf("%s: not equal (%v != %v)", msg, n1, n2)
	}
}

func (as *Assert) EqualIndent(n1, n2 any, msg string) {
	if !reflect.DeepEqual(n1, n2) {
		as.Errorf("%s: not equal \n%s\n%s", msg, n1, n2)
	}
}

func (as *Assert) integer(n1, n2 int, equal bool) bool {
	if equal {
		return n1 != n2
	}

	return n1 == n2
}

func (as *Assert) str(n1, n2 string, equal bool) bool {
	if equal {
		return n1 != n2
	}

	return n1 == n2
}

func (as *Assert) boolean(n1, n2, equal bool) bool {
	if equal {
		return n1 != n2
	}

	return n1 == n2
}

func (as *Assert) ptr(n1, n2 any, equal bool) bool {
	ptr1 := reflect.ValueOf(n1).Pointer()
	ptr2 := reflect.ValueOf(n2).Pointer()

	if equal {
		return ptr1 != ptr2
	}

	return ptr1 == ptr2
}

type TestHandler string

func (TestHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}

func (t *tree) String() string {
	bytes, err := json.MarshalIndent(t.root, "", "\t")
	if err != nil {
		panic(err) // unexpected
	}

	return string(bytes)
}

func (n *node) String() string {
	bytes, err := json.MarshalIndent(n, "", "\t")
	if err != nil {
		panic(err) // unexpected
	}

	return string(bytes)
}

func must(err error) {
	if err != nil {
		panic("unexpected error " + err.Error())
	}
}

func mustReq(req *http.Request, err error) *http.Request {
	must(err)

	return req
}
