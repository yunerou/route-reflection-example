// Command striptags rewrites Go struct tags, removing chosen keys (e.g. doc,
// example) and builds via `go build -overlay`, WITHOUT touching your source
// tree. Original files stay intact; modified copies live in a temp dir and are
// only fed to this one build.
//
//	go run striptags.go -drop doc,example -root . -o ./app ./cmd/app
//
// Omit -o to just print the overlay path and the go command to run yourself.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	root := flag.String("root", ".", "module root to scan")
	dropCSV := flag.String("drop", "doc,example", "comma-separated tag keys to remove")
	out := flag.String("o", "", "output binary path (if set, runs go build)")
	flag.Parse()
	target := flag.Arg(0) // package to build, e.g. ./cmd/app

	drop := map[string]bool{}
	for k := range strings.SplitSeq(*dropCSV, ",") {
		if k = strings.TrimSpace(k); k != "" {
			drop[k] = true
		}
	}

	tmp, err := os.MkdirTemp("", "striptags-")
	if err != nil {
		fatal(err)
	}

	replace := map[string]string{}
	err = filepath.WalkDir(*root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "testdata" || (strings.HasPrefix(name, ".") && name != ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		newSrc, changed, err := stripFile(path, drop)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if !changed {
			return nil
		}
		abs, _ := filepath.Abs(path)
		dst := filepath.Join(tmp, strings.ReplaceAll(abs, string(os.PathSeparator), "_"))
		if err := os.WriteFile(dst, newSrc, 0o644); err != nil {
			return err
		}
		replace[abs] = dst
		return nil
	})
	if err != nil {
		fatal(err)
	}

	overlayPath := filepath.Join(tmp, "overlay.json")
	blob, _ := json.MarshalIndent(struct{ Replace map[string]string }{replace}, "", "  ")
	if err := os.WriteFile(overlayPath, blob, 0o644); err != nil {
		fatal(err)
	}
	fmt.Fprintf(os.Stderr, "rewrote %d file(s); overlay: %s\n", len(replace), overlayPath)

	if *out == "" {
		fmt.Printf("go build -overlay %s -o app %s\n", overlayPath, target)
		return
	}
	args := []string{"build", "-overlay", overlayPath, "-o", *out}
	if target != "" {
		args = append(args, target)
	}
	cmd := exec.Command("go", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fatal(err)
	}
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "error:", err); os.Exit(1) }

// stripFile parses one file and splices out the dropped tag keys at the byte
// level (no full re-print), so everything else stays byte-for-byte identical.
func stripFile(path string, drop map[string]bool) ([]byte, bool, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, false, err
	}

	type edit struct {
		start, end int
		repl       string
	}
	var edits []edit

	ast.Inspect(f, func(n ast.Node) bool {
		st, ok := n.(*ast.StructType)
		if !ok || st.Fields == nil {
			return true
		}
		for _, fld := range st.Fields.List {
			if fld.Tag == nil {
				continue
			}
			raw, err := strconv.Unquote(fld.Tag.Value)
			if err != nil {
				continue
			}
			kept, changed := filterTag(raw, drop)
			if !changed {
				continue
			}
			repl := ""
			if kept != "" {
				repl = "`" + kept + "`"
			}
			edits = append(edits, edit{
				start: fset.Position(fld.Tag.Pos()).Offset,
				end:   fset.Position(fld.Tag.End()).Offset,
				repl:  repl,
			})
		}
		return true
	})
	if len(edits) == 0 {
		return src, false, nil
	}

	// Apply high offset -> low offset so earlier offsets stay valid.
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	res := src
	for _, e := range edits {
		res = append(res[:e.start:e.start], append([]byte(e.repl), res[e.end:]...)...)
	}
	return res, true, nil
}

// filterTag walks a struct tag using the same grammar as reflect.StructTag,
// keeping the order/spacing of surviving keys.
func filterTag(tag string, drop map[string]bool) (string, bool) {
	var parts []string
	changed := false
	rest := tag
	for {
		i := 0
		for i < len(rest) && rest[i] == ' ' {
			i++
		}
		rest = rest[i:]
		if rest == "" {
			break
		}
		i = 0
		for i < len(rest) && rest[i] > ' ' && rest[i] != ':' && rest[i] != '"' && rest[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(rest) || rest[i] != ':' || rest[i+1] != '"' {
			break // malformed tail: stop, keep nothing more
		}
		key := rest[:i]
		rest = rest[i+1:] // now at the opening quote
		i = 1
		for i < len(rest) && rest[i] != '"' {
			if rest[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(rest) {
			break
		}
		qval := rest[:i+1] // value including quotes
		rest = rest[i+1:]
		if drop[key] {
			changed = true
			continue
		}
		parts = append(parts, key+":"+qval)
	}
	return strings.Join(parts, " "), changed
}
