package gomux

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
		Formats:      []muxrouter.RegisterFormat{{Headers: muxrouter.JsonHeaders, Formats: muxrouter.JsonSonicFormat}},
		ConvertError: func(err error) muxrouter.StatusError { return statusErr{code: 500, msg: err.Error()} },
	}
}

type getParam struct {
	ID string `path:"id"`
}
type getResp struct {
	ID string `json:"id"`
}

func TestGomux_GetWithPathParam(t *testing.T) {
	r := New(testConfig())
	g := r.Create("/users")
	RegisterRoute(g, "GET", "/{id}", muxrouter.RouteMeta{},
		func(_ context.Context, p getParam, _ struct{}) (getResp, error) {
			return getResp{ID: p.ID}, nil
		}, nil)

	srv := httptest.NewServer(r.ExtractHandler(false))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/users/42")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out getResp
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.ID != "42" {
		t.Fatalf("got %+v", out)
	}
}

type postBody struct {
	Name string `json:"name"`
}

func TestGomux_PostWithBody(t *testing.T) {
	r := New(testConfig())
	g := r.Create("/things")
	RegisterRoute(g, "POST", "/", muxrouter.RouteMeta{},
		func(_ context.Context, _ struct{}, b postBody) (postBody, error) {
			return b, nil
		}, nil)

	srv := httptest.NewServer(r.ExtractHandler(false))
	defer srv.Close()

	resp, err := srv.Client().Post(srv.URL+"/things", "application/json", strings.NewReader(`{"name":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(out), `"name":"x"`) {
		t.Fatalf("got %s", out)
	}
}
