package gomux

import (
	"net/http/httptest"
	"testing"
)

type qp struct {
	Limit int    `query:"limit"`
	Name  string `query:"name"`
}

func TestParseParams_Query(t *testing.T) {
	r := httptest.NewRequest("GET", "/?limit=5&name=bob", nil)
	got, err := parseParams[qp](r)
	if err != nil {
		t.Fatal(err)
	}
	if got.Limit != 5 || got.Name != "bob" {
		t.Fatalf("got %+v", got)
	}
}

func TestSetFieldValue_BadInt(t *testing.T) {
	r := httptest.NewRequest("GET", "/?limit=notanint", nil)
	if _, err := parseParams[qp](r); err == nil {
		t.Fatal("expected error for bad int")
	}
}
