// Package sample exists only to give the gen-route-info tests a
// realistic RegisterRoute callsite to inspect.
package sample

import (
	"context"

	reflectionmux "github.com/yunerou/niarb/pkg/reflection-mux"
)

// GetUserReq carries the URL inputs for the GetUser route.
type GetUserReq struct {
	// ID is the numeric identifier of the user.
	ID int `path:"id"`
	// Include selects related resources to inline.
	Include string `query:"include"`
}

// GetUserResp is the body returned by GetUser.
type GetUserResp struct {
	// Name is the human-readable display name.
	Name string
	// Address is where the user lives.
	Address Address
}

// Address is a nested struct to exercise recursive comment extraction.
type Address struct {
	City    string // City where the user lives.
	Country string // Country in ISO-3166 alpha-2 form.
}

type appError struct{ msg string }

func (a appError) Error() string { return a.msg }

func register(mux reflectionmux.ReflectionMux) {
	reflectionmux.RegisterRoute[GetUserReq, struct{}, GetUserResp, appError](
		mux,
		"GET",
		"/users/{id}",
		reflectionmux.RouteMeta{Summary: "Get a user"},
		func(ctx context.Context, p GetUserReq, b struct{}) (GetUserResp, appError) {
			return GetUserResp{}, appError{}
		},
	)
}
