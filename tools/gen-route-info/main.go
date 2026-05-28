// Command gen-route-info inspects callsites of
// reflectionmux.RegisterRoute throughout one or more packages and
// produces a Go file that populates reflectionmux.CommentExtractor
// with the field comments of every struct type referenced by the
// generic type arguments (ReqParamT, ReqBodyT, RespBodyT).
//
// The generated file is consumed by reflectionmux at runtime to
// enrich RouteInfo with documentation that cannot be obtained
// through reflection alone.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	pathsFlag  = flag.String("paths", "", "Comma-separated package patterns to scan (e.g. ./..., ./app/...)")
	outputFlag = flag.String("output", "", "Path to the generated output file")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of gen-route-info:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if *pathsFlag == "" || *outputFlag == "" {
		fmt.Fprintln(os.Stderr, "both --paths and --output are required")
		usage()
		os.Exit(1)
	}

	patterns := splitPatterns(*pathsFlag)
	pkgs, err := loadPackages(patterns)
	if err != nil {
		log.Fatalf("load packages: %v", err)
	}

	calls := findRegisterRouteCalls(pkgs)
	log.Printf("found %d RegisterRoute call(s)", len(calls))

	structDocs := extractStructDocs(pkgs, calls)
	log.Printf("collected comments for %d struct type(s)", len(structDocs))

	if err := writeGenerated(*outputFlag, structDocs); err != nil {
		log.Fatalf("write generated file: %v", err)
	}
	log.Printf("wrote %s", *outputFlag)
}

func splitPatterns(s string) []string {
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
