package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// checkout writes a small tree shaped like the ephesus repo, with a secret
// beside it, and returns the handler serving the tree.
func checkout(t *testing.T) http.Handler {
	t.Helper()
	top := t.TempDir()
	dir := filepath.Join(top, "repo")
	files := map[string]string{
		"README.md": "> epigraph\n\n# Prerequisites\n\n<img src=\"./images/a.png\">\n\n" +
			"See [sub](./sub/) and [other](./OTHER.md).\n\n## US Federal Reserve\n\ntext\n",
		"OTHER.md":             "# Other & More\n\nbody\n",
		"NOHEAD.md":            "plain text only\n",
		"images/a.png":         "PNG",
		"sub/README.md":        "# Sub page\n\n<img src=\"./images/b.png\">\n\n[back](../README.md#us-federal-reserve)\n",
		"sub/images/b.png":     "PNGB",
		"bare/tools/x.py":      "print(1)\n",
		"bare/notes.txt":       "notes\n",
		"files/saved.html":     "<script>alert(1)</script>",
		"files/a b.pdf":        "%PDF",
		"sp ace/README.md":     "# Spaced\n",
		".git/config":          "[core]\n",
		".gitignore":           "bin/\n",
		"bare/.DS_Store":       "junk",
		"bare/tools/.hidden":   "junk",
		"jesus_and_taxes/x.md": "# X\n",
	}
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(top, "outside.txt"), []byte("SECRET"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../outside.txt", filepath.Join(dir, "escape.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("OTHER.md", filepath.Join(dir, "alias.md")); err != nil {
		t.Fatal(err)
	}
	h, err := newHandler(dir)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func do(h http.Handler, method, target string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, target, nil))
	return w
}

func TestRoutes(t *testing.T) {
	h := checkout(t)
	for _, tc := range []struct {
		target   string
		code     int
		location string   // for redirects
		has      []string // in the body
		header   [2]string
	}{
		{target: "/", code: 200, has: []string{
			"<title>Main - Ephesus</title>",
			`<h1 id="prerequisites">Prerequisites</h1>`,
			`<img src="./images/a.png">`,
			`<a href="./OTHER.md">other</a>`,
			`<li><a href="#us-federal-reserve">US Federal Reserve</a>`,
			`<li class="active"><a href="/">Main</a></li>`,
			`<link rel="stylesheet" href="/static/css/style.css">`,
		}},
		{target: "/images/a.png", code: 200, has: []string{"PNG"}},
		{target: "/README.md", code: 301, location: "/"},
		{target: "/README.md?x=1", code: 301, location: "/?x=1"},
		{target: "/sub", code: 301, location: "/sub/"},
		{target: "/sub/", code: 200, has: []string{"Sub page", `<img src="./images/b.png">`, "../README.md#us-federal-reserve"}},
		{target: "/sub/README.md", code: 301, location: "/sub/"},
		{target: "/sub/images/b.png", code: 200, has: []string{"PNGB"}},
		{target: "/jesus_and_taxes", code: 301, location: "/jesus_and_taxes/"},
		{target: "/OTHER.md", code: 200, has: []string{
			"<title>Other &amp; More - Ephesus</title>",
			`<a href="#other-more">Other &amp; More</a>`,
		}},
		{target: "/OTHER.md/", code: 301, location: "/OTHER.md"},
		{target: "/alias.md", code: 200, has: []string{"Other &amp; More"}},
		{target: "/NOHEAD.md", code: 200, has: []string{"<title>NOHEAD - Ephesus</title>", `class="page-nav"`, "plain text only"}},
		{target: "/sp%20ace", code: 301, location: "/sp%20ace/"},
		{target: "/sp%20ace/", code: 200, has: []string{"Spaced"}},
		{target: "/files/a%20b.pdf", code: 200, has: []string{"%PDF"}},
		{target: "/bare/", code: 200, has: []string{`<a href="tools/">tools/</a>`, `<a href="notes.txt">notes.txt</a>`, `<a href="../">../</a>`}},
		{target: "/bare/tools/x.py", code: 200, header: [2]string{"Content-Type", "text/plain; charset=utf-8"}},
		{target: "/bare/notes.txt", code: 200, header: [2]string{"Content-Type", "text/plain; charset=utf-8"}},
		{target: "/files/saved.html", code: 200, header: [2]string{"Content-Security-Policy", "sandbox"}},
		{target: "/static/css/style.css", code: 200, has: []string{"#toc"}},
		{target: "/.git/config", code: 404},
		{target: "/.git/", code: 404},
		{target: "/.gitignore", code: 404},
		{target: "/bare/.DS_Store", code: 404},
		{target: "/escape.txt", code: 404},
		{target: "/nope", code: 404},
	} {
		w := do(h, "GET", tc.target)
		if w.Code != tc.code {
			t.Errorf("GET %s: %d, want %d", tc.target, w.Code, tc.code)
			continue
		}
		if tc.location != "" && w.Header().Get("Location") != tc.location {
			t.Errorf("GET %s: Location %q, want %q", tc.target, w.Header().Get("Location"), tc.location)
		}
		body := w.Body.String()
		for _, s := range tc.has {
			if !strings.Contains(body, s) {
				t.Errorf("GET %s: body lacks %q", tc.target, s)
			}
		}
		if k := tc.header[0]; k != "" && w.Header().Get(k) != tc.header[1] {
			t.Errorf("GET %s: %s %q, want %q", tc.target, k, w.Header().Get(k), tc.header[1])
		}
		if strings.Contains(body, "SECRET") || strings.Contains(body, "[core]") || strings.Contains(body, ".DS_Store") {
			t.Errorf("GET %s: body shows what it must not:\n%s", tc.target, body)
		}
	}
}

// Paths that climb out of the checkout end at a 404 or a redirect, never at a
// file outside it.
func TestNoEscape(t *testing.T) {
	h := checkout(t)
	for _, target := range []string{"/../outside.txt", "/sub/../../outside.txt", "/%2e%2e/outside.txt", "/escape.txt"} {
		w := do(h, "GET", target)
		if w.Code == 200 || strings.Contains(w.Body.String(), "SECRET") {
			t.Errorf("GET %s: %d %q", target, w.Code, w.Body.String())
		}
	}
	// The site handler itself, past the mux's path cleaning.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.URL.Path = "/../outside.txt"
	siteOf(t, h).ServeHTTP(w, r)
	if w.Code == 200 || strings.Contains(w.Body.String(), "SECRET") {
		t.Errorf("site GET /../outside.txt: %d %q", w.Code, w.Body.String())
	}
}

func siteOf(t *testing.T, h http.Handler) *site {
	t.Helper()
	got, _ := h.(*http.ServeMux).Handler(httptest.NewRequest("GET", "/x", nil))
	s, ok := got.(*site)
	if !ok {
		t.Fatalf("handler for /x is %T", got)
	}
	return s
}

func TestMethods(t *testing.T) {
	h := checkout(t)
	if w := do(h, "POST", "/"); w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /: %d", w.Code)
	}
	if w := do(h, "HEAD", "/images/a.png"); w.Code != 200 {
		t.Errorf("HEAD /images/a.png: %d", w.Code)
	}
}

func TestNotACheckout(t *testing.T) {
	if _, err := newHandler(t.TempDir()); err == nil {
		t.Error("a folder without README.md was accepted")
	}
}

// The table of contents stays well formed whatever the heading levels.
func TestTOCBalanced(t *testing.T) {
	for _, md := range []string{
		"## a\n### b\n# c\n## d\n",
		"### a\n# b\n",
		"# a\n### b\n## c\n",
		"no headings\n",
	} {
		toc := buildTOC(parseMarkdown([]byte(md)))
		if o, c := strings.Count(toc, "<ul>"), strings.Count(toc, "</ul>"); o != c {
			t.Errorf("%q: %d <ul>, %d </ul>: %s", md, o, c, toc)
		}
	}
}
