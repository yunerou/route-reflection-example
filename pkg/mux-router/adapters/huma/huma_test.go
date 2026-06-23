package huma

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	muxrouter "github.com/yunerou/niarb/pkg/mux-router"
)

type statusErr struct {
	code int
	msg  string
}

func (e statusErr) GetStatus() int { return e.code }
func (e statusErr) Error() string  { return e.msg }

func testConfig() muxrouter.Config {
	return muxrouter.Config{
		ConvertError: func(err error) muxrouter.StatusError { return statusErr{500, err.Error()} },
		Doc:          muxrouter.DocConfig{Title: "Test", Version: "1.0.0", DocsPath: "/docs", OpenAPIPath: "/openapi"},
	}
}

type getParam struct {
	ID string `path:"id"`
}
type getResp struct {
	ID string `json:"id"`
}

func TestHuma_GetWithPathParam(t *testing.T) {
	r := New(testConfig())
	g := r.Create("/users")
	RegisterRoute(g, "GET", "/{id}", muxrouter.RouteMeta{},
		func(_ context.Context, p getParam, _ struct{}) (getResp, error) {
			return getResp{ID: p.ID}, nil
		}, nil)

	srv := httptest.NewServer(r.ExtractHandler(true))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/users/42")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.ID != "42" {
		t.Fatalf("got %+v", out)
	}
}

func TestHuma_OpenAPIServedWhenDocEnabled(t *testing.T) {
	r := New(testConfig())
	g := r.Create("/users")
	RegisterRoute(g, "GET", "/{id}", muxrouter.RouteMeta{Summary: "get"},
		func(_ context.Context, p getParam, _ struct{}) (getResp, error) {
			return getResp{ID: p.ID}, nil
		}, nil)
	srv := httptest.NewServer(r.ExtractHandler(true))
	defer srv.Close()

	resp, _ := srv.Client().Get(srv.URL + "/openapi.json")
	if resp.StatusCode != 200 {
		t.Fatalf("expected openapi 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "/users/{id}") {
		t.Fatalf("openapi missing route: %s", body)
	}
	// The path parameter must be documented as an OpenAPI parameter.
	if !strings.Contains(string(body), `"in":"path"`) || !strings.Contains(string(body), `"name":"id"`) {
		t.Fatalf("openapi missing path param doc: %s", body)
	}
}
