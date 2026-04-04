package mixer

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

type (
	// tree represents the tree data structure that contains parts of URL.
	// insert operation: /a/b/c/ URL into nodes a, b, c, /
	// search operation: depth-first search.
	tree struct {
		root *node
	}

	// node represents the set of http.Handler and can be "typed".
	// Different nodes obey the next rules:
	//   `*` | `:` | `/`  , where `:` - path param, `/` - trailing slash, `*` - other
	//  0)  0  |  0  |  0  -> node ready to be set
	//  1)  0  |  1  |  0  -> only one `:` per node
	//  2)  0  |  1  |  1  -> combination `:` and `/` allowed
	//  3)  1  |  0  |  0  -> any combination of `*` per node
	//  4)  1  |  0  |  1  -> combination `*` and `/` allowed
	//  5)  1  |  1  |  0  -> `*` and `:` together allowed; static match has priority
	//  6)  1  |  1  |  1  -> `*`, `:`, `/` together allowed
	node struct {
		conv     *convert
		Methods  map[string]http.Handler `json:"methods"`
		Children map[string]*node        `json:"children"`
		tid      int
	}

	// convert represents the convert function for path params.
	convert func(string) (any, error)

	// contextKey is a value for use with context.WithValue.
	contextKey string
)

type invalidIntError struct {
	val string
}

func (e invalidIntError) Error() string {
	return fmt.Sprintf("mixer: invalid int %q", e.val)
}

const (
	other = iota // other `*`
	param        // path param `:`
	slash        // trailing slash `/`
	root         // only for the tree.root node
)

const (
	// pathToken determines delimiter for splitting URL parts.
	pathToken = "/"

	// typeToken determines a special token for URL path params.
	typeToken = ":"
)

// methodError wraps the ErrMethod error.
func methodError(m, p string) *ServeMuxError {
	return &ServeMuxError{err: ErrMethod, method: m, pattern: p}
}

// handlerError wraps the ErrHandler error.
func handlerError(m, p string) *ServeMuxError {
	return &ServeMuxError{err: ErrHandler, method: m, pattern: p}
}

// patternError wraps the ErrPattern error.
func patternError(m, p string) *ServeMuxError {
	return &ServeMuxError{ErrPattern, m, p}
}

// duplicateError wraps the ErrDuplicate error.
func duplicateError(m, p string) *ServeMuxError {
	return &ServeMuxError{ErrDuplicate, m, p}
}

// notFoundError wraps the ErrNotFound error.
func notFoundError(m, p string) *ServeMuxError {
	return &ServeMuxError{ErrNotFound, m, p}
}

// intConv adapts the interface of the type conversion function from string to int.
func intConv(s string) (any, error) {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0, invalidIntError{s}
	}

	return val, nil
}

// strConv adapts the interface of the type conversion function from string to string.
func strConv(s string) (any, error) {
	return s, nil
}

// splitURL splits incoming url to parts separated by pathToken.
// Any trailing slash will be a part too. The root path is ignored.
// If an error occurred, parts will return anyway.
func splitURL(url string) ([]string, error) {
	if !strings.HasPrefix(url, pathToken) {
		return []string{}, ErrPattern
	}

	parts := strings.Split(url[1:], pathToken)

	if parts[len(parts)-1] == "" {
		parts[len(parts)-1] = pathToken
	}

	if slices.Contains(parts, "") {
		return parts, ErrPattern
	}

	return parts, nil
}

// deepcopy returns a full copy of the receiver tree.
// For conv and Methods stores only links because if
// the insert operation was correct, copy can replace the origin.
func (t *tree) deepcopy() *tree {
	// t.root == nil is unexpected
	return &tree{root: t.root.deepcopy()}
}

// deepcopy returns a full copy from the receiver node.
// For conv and Methods stores only links because if
// the insert operation was correct, copy can replace the origin.
func (n *node) deepcopy() *node {
	// n == nil is unexpected
	nodeCopy := new(node)
	*nodeCopy = *n

	if n.Children != nil {
		nodeCopy.Children = make(map[string]*node)
	}

	for k, v := range n.Children {
		nodeCopy.Children[k] = v.deepcopy()
	}

	return nodeCopy
}

// find finds child node by type ID.
func (n *node) find(tid int) *node {
	for _, c := range n.Children {
		if c.tid == tid {
			return c
		}
	}

	return nil
}

// insert inserts a new node starting from n.
func (n *node) insert(key string, in *node) {
	if n.Children == nil {
		n.Children = make(map[string]*node)
	}

	n.Children[key] = in
}

// insert builds parts to the inner tree by some rules:
//   - if the node for the part does not exist, it will be created.
//   - if the node for a part exists, it will be returned.
//
// Returns methods associated with the last inserted or found node.
func (mux *ServeMux) insert(parts []string) (map[string]http.Handler, error) {
	treeCopy := mux.tree.deepcopy()
	curr := treeCopy.root

	for _, part := range parts {
		newNode := new(node)

		switch part[:1] {
		case pathToken:
			newNode.tid = slash
		case typeToken:
			conv := mux.converters[part[1:]]

			if conv == nil {
				return nil, ErrPathParam
			}

			newNode.tid = param
			newNode.conv = conv

			part = typeToken
		}

		child, ok := curr.Children[part]
		if ok && child.conv != newNode.conv {
			return nil, ErrMultiplePathParam
		}

		if ok {
			curr = child

			continue
		}

		curr.insert(part, newNode)
		curr = newNode
	}

	if curr.Methods == nil {
		curr.Methods = make(map[string]http.Handler)
	}

	*mux.tree = *treeCopy

	return curr.Methods, nil
}
