package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEndToEnd loads the bundled fixture, runs the full pipeline and
// asserts the generated output contains every expected struct and
// every expected field comment.
func TestEndToEnd(t *testing.T) {
	t.Parallel()

	const fixturePath = "./testdata/sample"
	pkgs, err := loadPackages([]string{fixturePath})
	if err != nil {
		t.Fatalf("loadPackages: %v", err)
	}

	calls := findRegisterRouteCalls(pkgs)
	if len(calls) != 1 {
		t.Fatalf("expected 1 RegisterRoute call in fixture, got %d", len(calls))
	}

	docs := extractStructDocs(pkgs, calls)
	gotTypes := map[string]map[string]string{}
	for _, d := range docs {
		gotTypes[d.TypeName] = d.FieldDocs
	}

	wantTypes := map[string]map[string]string{
		"GetUserReq": {
			"ID":      "ID is the numeric identifier of the user.",
			"Include": "Include selects related resources to inline.",
		},
		"GetUserResp": {
			"Name":    "Name is the human-readable display name.",
			"Address": "Address is where the user lives.",
		},
		"Address": {
			"City":    "City where the user lives.",
			"Country": "Country in ISO-3166 alpha-2 form.",
		},
	}
	for name, want := range wantTypes {
		got, ok := gotTypes[name]
		if !ok {
			t.Errorf("missing struct %s in output", name)
			continue
		}
		for f, doc := range want {
			if got[f] != doc {
				t.Errorf("%s.%s: want %q, got %q", name, f, doc, got[f])
			}
		}
	}

	out := filepath.Join(t.TempDir(), "out.go")
	if err := writeGenerated(out, docs); err != nil {
		t.Fatalf("writeGenerated: %v", err)
	}
	content, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	for _, snippet := range []string{
		"package reflectionmux",
		`"reflect"`,
		"reflect.TypeFor[sample.GetUserReq]()",
		"reflect.TypeFor[sample.GetUserResp]()",
		"reflect.TypeFor[sample.Address]()",
		`"ID is the numeric identifier of the user."`,
		`"City where the user lives."`,
	} {
		if !strings.Contains(string(content), snippet) {
			t.Errorf("generated output missing snippet: %s\n---\n%s", snippet, content)
		}
	}
}

func TestSplitPatterns(t *testing.T) {
	t.Parallel()
	got := splitPatterns(" ./..., ./app/server , ")
	want := []string{"./...", "./app/server"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
