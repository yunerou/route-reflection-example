package muxrouter

import (
	"reflect"
	"testing"
)

type okParam struct {
	ID string `path:"id"`
}
type mismatchParam struct {
	Other string `path:"other"`
}

func typeFor[T any]() reflect.Type { return reflect.TypeFor[T]() }

func TestValidateRoute_OK(t *testing.T) {
	info := ValidateRoute[okParam, struct{}, struct{}]("GET", "/users/{id}")
	if info.ReqParamType.Name() != "okParam" {
		t.Fatalf("expected okParam, got %s", info.ReqParamType.Name())
	}
}

func TestValidateRoute_MismatchPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on path param mismatch")
		}
	}()
	ValidateRoute[mismatchParam, struct{}, struct{}]("GET", "/users/{id}")
}

func TestTypeToSchema_Struct(t *testing.T) {
	type body struct {
		Name string `json:"name"`
	}
	got := string(TypeToSchema(typeFor[body]()))
	want := `{"name":"string"}`
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
