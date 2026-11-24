package sarvam

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChooseAndName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("api-subscription-key") != "k" {
			http.Error(w, "no", 403)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch r.URL.Path {
		case "/v2/systemone":
			q := body["questions"].(map[string]any)["pick"].(map[string]any)
			if q["type"] != "choice" || len(q["criteria"].(map[string]any)) != 2 || body["model"] != "sarvam-prism" {
				http.Error(w, "bad request", 422)
				return
			}
			_, _ = w.Write([]byte(`{"answers":{"pick":{"type":"choice","choice":"a","probabilities":{"a":0.9,"NONE":0.1}}}}`))
		case "/v1/chat/completions":
			_, _ = w.Write([]byte("{\"choices\":[{\"message\":{\"content\":\"```json\\n{\\\"title\\\":\\\"Water leak\\\",\\\"description\\\":\\\"A leak.\\\"}\\n```\"}}]}"))
		}
	}))
	defer srv.Close()
	c := New("k", srv.URL, "small")
	p, err := c.Choose(context.Background(), "msg", "which?", []Option{{ID: "a", Text: "A"}, {ID: "NONE", Text: "none"}})
	if err != nil || p["a"] != 0.9 {
		t.Fatalf("%v %v", p, err)
	}
	n, err := c.Name(context.Background(), "paani leak")
	if err != nil || n.Title != "Water leak" || n.Description != "A leak." {
		t.Fatalf("%+v %v", n, err)
	}
	if _, err := New("wrong", srv.URL, "small").Choose(context.Background(), "x", "y", nil); err == nil {
		t.Fatal("expected auth error")
	}
}
