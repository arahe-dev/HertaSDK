package retrycheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestAnalyzer(t *testing.T) {
	src := `package sample
import (
	h "net/http"
	r "github.com/hashicorp/go-retryablehttp"
	id "github.com/google/uuid"
	retry "github.com/avast/retry-go/v4"
)
func request() {
	stable := id.NewString()
	for attempt := 0; attempt < 4; attempt++ {
		req.Header.Set("Idempotency-Key", stable)
		local := id.NewString()
		req.Header.Set("Idempotency-Key", local) // RK001
		req.Header.Add("X-Idempotency-Key", id.New().String()) // RK001
		params.SetIdempotencyKey(newIdempotencyKey()) // RK001
		opts.IdempotencyKey = id.NewString() // RK001
		opts = Options{IdempotencyKey: id.NewString()} // RK001
		later := func() {
			req.Header.Set("Idempotency-Key", id.NewString())
		}
	}
	for _, customer := range customers {
		charge(customer, Options{IdempotencyKey: id.NewString()})
	}
	retry.Do(func() error {
		req.Header.Set("PayPal-Request-Id", id.NewString()) // RK001
		return do()
	})
}
func send() {
	r.Post(url, "text/plain", body) // RK002
	r.NewClient().StandardClient().PostForm(url, values) // RK002
	client := r.NewClient()
	req, _ := r.NewRequest("POST", url, body)
	client.Do(req) // RK002
	patch, _ := h.NewRequestWithContext(ctx, h.MethodPatch, url, body)
	client.Do(patch) // RK002
	req.Header.Set("Idempotency-Key", key)
	client.Do(req)
	read, _ := r.NewRequest("GET", url, nil)
	client.Do(read)
	other := h.DefaultClient
	other.Post(url, "text/plain", body)
	client = other
	client.Post(url, "text/plain", body)
}
func (c *Client) Do(req *h.Request) { // RK002
	for attempt := 0; attempt < 4; attempt++ {
		c.HTTP.Do(req)
	}
}
func (c *Client) RoundTrip(req *h.Request) {
	if req.Method == "POST" {
		c.sendOnce(req)
		return
	}
	for attempt := 0; attempt < 4; attempt++ {
		c.HTTP.RoundTrip(req)
	}
}
func (c *NestedClient) Do(req *h.Request) { // RK002
	if c.enabled {
		for attempt := 0; attempt < 4; attempt++ {
			c.HTTP.Do(req)
		}
	}
}
func (c *CallbackClient) Do(req *h.Request) {
	save(func() {
		for attempt := 0; attempt < 4; attempt++ {
			c.HTTP.Do(req)
		}
	})
}
func receiveWebhook(w h.ResponseWriter, req *h.Request) {
	event := webhook.ConstructEvent(req)
	queue.Publish(event) // RK003
}
func receiveSafeWebhook(w h.ResponseWriter, req *h.Request) {
	event := webhook.ConstructEvent(req)
	if !store.ClaimEvent(event.ID) {
		return
	}
	queue.Publish(event)
}
func addRoute() {
	router.Handle("/webhooks/provider", func(w h.ResponseWriter, req *h.Request) {
		mail.Send(req) // RK003
	})
}
func pingWebhook(w h.ResponseWriter, req *h.Request) {
	w.Write([]byte("pong"))
}
func sendWebhook(url string) {
	client.Send(url)
}
func unrelated(w h.ResponseWriter, req *h.Request) {
	queue.Publish(req)
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "sample.go", src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}

	var got, want []string
	_, err = Analyzer.Run(&analysis.Pass{
		Analyzer: Analyzer,
		Fset:     fset,
		Files:    []*ast.File{f},
		Report: func(d analysis.Diagnostic) {
			got = append(got, strconv.Itoa(fset.Position(d.Pos).Line)+": "+d.Category)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantPattern := regexp.MustCompile(`// (RK00[123])$`)
	for i, line := range strings.Split(src, "\n") {
		if match := wantPattern.FindStringSubmatch(line); match != nil {
			want = append(want, strconv.Itoa(i+1)+": "+match[1])
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("findings = %v; want %v", got, want)
	}
}
