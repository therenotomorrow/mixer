package mixer_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/therenotomorrow/mixer"
)

type Item struct {
	Name string `json:"name,omitempty"`
	ID   int    `json:"id,omitempty"`
}

type Catalog struct {
	items map[int]*Item
	ident int
	mutex sync.RWMutex
}

func JSONResponse[T *Item | []*Item](writer http.ResponseWriter, code int, body T) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)

	if body == nil {
		_, _ = fmt.Fprintln(writer, "{}")

		return
	}

	err := json.NewEncoder(writer).Encode(body)
	if err != nil {
		slog.Error("cannot encode body", "error", err, "body", body)
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}
}

func NewCatalog() *Catalog {
	return &Catalog{
		mutex: sync.RWMutex{},
		items: make(map[int]*Item),
		ident: 0,
	}
}

func (c *Catalog) Get(id int) *Item {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.items[id]
}

func (c *Catalog) Add(name string) *Item {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.ident++
	c.items[c.ident] = &Item{ID: c.ident, Name: name}

	return c.items[c.ident]
}

func (c *Catalog) List() []*Item {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	items := slices.Collect(maps.Values(c.items))

	slices.SortFunc(items, func(a, b *Item) int {
		return a.ID - b.ID
	})

	return items
}

func (c *Catalog) All(w http.ResponseWriter, _ *http.Request) {
	items := c.List()

	JSONResponse(w, http.StatusOK, items)
}

func (c *Catalog) Create(w http.ResponseWriter, r *http.Request) {
	name := r.PostFormValue("name")
	item := c.Add(name)

	JSONResponse(w, http.StatusCreated, item)
}

func (c *Catalog) Retrieve(w http.ResponseWriter, r *http.Request) {
	id := mixer.GetPathParams(r)[0].(int)
	item := c.Get(id)

	JSONResponse(w, http.StatusOK, item)
}

func readAndPrint(resp *http.Response) {
	defer func() { _ = resp.Body.Close() }()

	buf := bytes.NewBuffer(nil)
	_, _ = buf.ReadFrom(resp.Body)

	fmt.Printf("code: %d, body: %s", resp.StatusCode, buf.String())
}

func ExampleNewServeMux() {
	catalog := NewCatalog()
	catalog.Add("Foo")
	catalog.Add("Bar")

	mux := mixer.New()
	srv := http.Server{Addr: ":8080", Handler: mux}

	mux.GetFunc("/catalog/", catalog.All)
	mux.PostFunc("/catalog/", catalog.Create)
	mux.GetFunc("/catalog/:int", catalog.Retrieve)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { _ = srv.ListenAndServe() }()

	go func() {
		var resp *http.Response

		time.Sleep(time.Second)

		resp, _ = http.Get("http://127.0.0.1:8080/catalog")
		readAndPrint(resp)

		resp, _ = http.Get("http://127.0.0.1:8080/catalog/")
		readAndPrint(resp)

		resp, _ = http.Post(
			"http://127.0.0.1:8080/catalog/",
			"application/x-www-form-urlencoded",
			bytes.NewBufferString("name=Mem"),
		)
		readAndPrint(resp)

		resp, _ = http.Get("http://127.0.0.1:8080/catalog/")
		readAndPrint(resp)

		resp, _ = http.Get("http://127.0.0.1:8080/catalog/3")
		readAndPrint(resp)

		resp, _ = http.Get("http://127.0.0.1:8080/catalog/4")
		readAndPrint(resp)

		resp, _ = http.Post(
			"http://127.0.0.1:8080/catalog/4",
			"application/x-www-form-urlencoded",
			bytes.NewBufferString("name=Kek"),
		)
		readAndPrint(resp)

		cancel()
	}()

	<-ctx.Done()
	_ = srv.Shutdown(ctx)

	// Output:
	// code: 404, body: 404 page not found
	// code: 200, body: [{"name":"Foo","id":1},{"name":"Bar","id":2}]
	// code: 201, body: {"name":"Mem","id":3}
	// code: 200, body: [{"name":"Foo","id":1},{"name":"Bar","id":2},{"name":"Mem","id":3}]
	// code: 200, body: {"name":"Mem","id":3}
	// code: 200, body: {}
	// code: 404, body: 404 page not found
}
