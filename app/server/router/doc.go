// Package router is the build-tag facade over the mux-router adapters.
//
// Default build selects the huma adapter (OpenAPI docs + validation).
// Build with -tags prod to select the gomux adapter (net/http only, smaller binary).
//
// Application code is identical across builds:
//
//	r := router.New(cfg)
//	g := r.Create("/users")
//	router.RegisterRoute(g, "GET", "/{id}", meta, handler, nil)
//	h := r.ExtractHandler(enableDoc)
package router
