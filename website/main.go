// Command website serves the ephesus repository as the site ephesus.church.
//
// Every Markdown file is rendered to a page (a folder's README.md is that
// folder's page) and every other file is served as it is, each at the path it
// has in the repository, so the relative links and <img src="./images/...">
// tags in the Markdown resolve as they do on GitHub. The files are read from
// the checkout on every request, so a git pull there is live at once.
//
//	website -content /srv/ephesus -listen 127.0.0.1:5001
//
// Run from website/ with no flags, it serves the checkout it sits in.
package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// The page template and the stylesheet are built into the binary; the content
// is read from the checkout.
//
//go:embed home.html.tmpl static
var assets embed.FS

var tmpl = template.Must(template.ParseFS(assets, "home.html.tmpl"))

type PageNav struct {
	Path   string
	Title  string
	Active bool
}

type PageData struct {
	Title   string
	Content template.HTML
	TOC     template.HTML
	Pages   []PageNav
}

// pages are the pages listed in the sidebar, each a folder whose README.md is
// the page. Every other Markdown file is served too, at its own path.
var pages = []struct {
	Path  string // URL path
	Title string // display title
}{
	{"/", "Main"},
	{"/usdollar/", "U.S. Dollar"},
	{"/jesus_and_taxes/", "Jesus & Taxes"},
	{"/revelation12/", "Revelation 12"},
}

func main() {
	listen := flag.String("listen", "127.0.0.1:5001", "address to serve on")
	content := flag.String("content", "..", "the ephesus checkout to serve (README.md at its top)")
	flag.Parse()

	h, err := newHandler(*content)
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{
		Addr:              *listen,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	log.Printf("serving %s on %s", *content, *listen)
	log.Fatal(srv.ListenAndServe())
}

// newHandler serves the checkout at dir, with the stylesheet under /static/.
func newHandler(dir string) (http.Handler, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	if _, err := root.Stat("README.md"); err != nil {
		return nil, fmt.Errorf("%s is not an ephesus checkout: %w", dir, err)
	}
	static, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	mux.Handle("/", &site{root: root, fsys: root.FS()})
	return mux, nil
}

// site serves the checkout: Markdown rendered, everything else as it is.
// os.Root keeps every lookup, symbolic links included, inside the checkout.
type site struct {
	root *os.Root
	fsys fs.FS
}

func (s *site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.Trim(path.Clean("/"+r.URL.Path), "/")
	if name == "" {
		name = "."
	}
	if hidden(name) {
		http.NotFound(w, r)
		return
	}
	fi, err := s.root.Stat(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	slash := strings.HasSuffix(r.URL.Path, "/")
	switch {
	case fi.IsDir():
		if !slash {
			// Relative links in a folder's README.md need the slash.
			redirect(w, r, "/"+name+"/")
			return
		}
		readme := path.Join(name, "README.md")
		if rfi, err := s.root.Stat(readme); err == nil && rfi.Mode().IsRegular() {
			s.servePage(w, readme)
		} else {
			s.serveListing(w, r, name)
		}
	case !fi.Mode().IsRegular():
		http.NotFound(w, r)
	case slash:
		redirect(w, r, "/"+name)
	case path.Base(name) == "README.md":
		// Cross-references to a README.md go to its folder's page.
		redirect(w, r, dirURL(name))
	case strings.EqualFold(path.Ext(name), ".md"):
		s.servePage(w, name)
	default:
		s.serveFile(w, r, name, fi)
	}
}

// hidden reports whether a path has a part that starts with a dot: .git,
// .gitignore, .DS_Store.
func hidden(name string) bool {
	for _, part := range strings.Split(name, "/") {
		if part != "." && strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

// dirURL is the URL of the folder that holds file: "/" or "/usdollar/".
func dirURL(file string) string {
	if d := path.Dir(file); d != "." {
		return "/" + d + "/"
	}
	return "/"
}

// redirect sends a permanent redirect to the URL path p, keeping the query.
func redirect(w http.ResponseWriter, r *http.Request, p string) {
	u := url.URL{Path: p, RawQuery: r.URL.RawQuery}
	http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
}

// servePage renders the Markdown file to a page.
func (s *site) servePage(w http.ResponseWriter, mdFile string) {
	md, err := fs.ReadFile(s.fsys, mdFile)
	if err != nil {
		http.Error(w, "Could not read markdown file", http.StatusInternalServerError)
		log.Println("error reading:", mdFile, err)
		return
	}
	doc := parseMarkdown(md)

	current := "/" + mdFile
	if path.Base(mdFile) == "README.md" {
		current = dirURL(mdFile)
	}
	title := ""
	for _, pg := range pages {
		if pg.Path == current {
			title = pg.Title
		}
	}
	if title == "" {
		title = firstHeading(doc)
	}
	if title == "" {
		if path.Base(mdFile) == "README.md" {
			title = path.Base(path.Dir(mdFile))
		} else {
			title = strings.TrimSuffix(path.Base(mdFile), path.Ext(mdFile))
		}
	}

	render(w, PageData{
		Title:   title,
		Content: template.HTML(renderMarkdown(doc)),
		TOC:     template.HTML(buildTOC(doc)),
		Pages:   pageNav(current),
	})
}

// serveListing lists a folder that has no README.md, as GitHub does
// (revelation12/README.md links to its tools/ and images/).
func (s *site) serveListing(w http.ResponseWriter, r *http.Request, dir string) {
	entries, err := fs.ReadDir(s.fsys, dir)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<h1>%s/</h1>\n<ul>\n<li><a href=\"../\">../</a></li>\n", template.HTMLEscapeString(dir))
	for _, e := range entries {
		n := e.Name()
		if strings.HasPrefix(n, ".") {
			continue
		}
		if e.IsDir() {
			n += "/"
		}
		href := (&url.URL{Path: n}).String()
		fmt.Fprintf(&b, "<li><a href=\"%s\">%s</a></li>\n", template.HTMLEscapeString(href), template.HTMLEscapeString(n))
	}
	b.WriteString("</ul>\n")
	render(w, PageData{
		Title:   dir + "/",
		Content: template.HTML(b.String()),
		Pages:   pageNav("/" + dir + "/"),
	})
}

// serveFile serves a file as it is. A saved web page or a drawing, which can
// carry scripts, is sandboxed so they never run as this site; source and text
// files are shown as UTF-8 text, as GitHub shows them.
func (s *site) serveFile(w http.ResponseWriter, r *http.Request, name string, fi fs.FileInfo) {
	f, err := s.root.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	switch strings.ToLower(path.Ext(name)) {
	case ".html", ".htm", ".xhtml", ".svg", ".xml":
		h.Set("Content-Security-Policy", "sandbox")
	case ".txt", ".py", ".go", ".sh", ".mdi", ".tmpl":
		h.Set("Content-Type", "text/plain; charset=utf-8")
	}
	http.ServeContent(w, r, name, fi.ModTime(), f)
}

func pageNav(current string) []PageNav {
	var nav []PageNav
	for _, pg := range pages {
		nav = append(nav, PageNav{
			Path:   pg.Path,
			Title:  pg.Title,
			Active: pg.Path == current,
		})
	}
	return nav
}

func render(w http.ResponseWriter, data PageData) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Println("template error:", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func parseMarkdown(md []byte) ast.Node {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	return parser.NewWithExtensions(extensions).Parse(md)
}

func renderMarkdown(doc ast.Node) []byte {
	opts := html.RendererOptions{
		Flags: html.CommonFlags,
	}
	return markdown.Render(doc, html.NewRenderer(opts))
}

// buildTOC generates an HTML table of contents from markdown headings.
func buildTOC(doc ast.Node) string {
	var buf bytes.Buffer
	prevLevel := 0
	open := 0 // lists open

	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		heading, ok := node.(*ast.Heading)
		if !ok || !entering {
			return ast.GoToNext
		}
		level := heading.Level

		if open == 0 {
			buf.WriteString("<ul>")
			open = 1
		} else if level > prevLevel {
			for i := prevLevel; i < level; i++ {
				buf.WriteString("<ul>")
				open++
			}
		} else if level < prevLevel {
			for i := level; i < prevLevel && open > 1; i++ {
				buf.WriteString("</li></ul>")
				open--
			}
			buf.WriteString("</li>")
		} else {
			buf.WriteString("</li>")
		}

		fmt.Fprintf(&buf, `<li><a href="#%s">%s</a>`,
			template.HTMLEscapeString(heading.HeadingID), template.HTMLEscapeString(extractText(heading)))
		prevLevel = level
		return ast.GoToNext
	})

	for ; open > 0; open-- {
		buf.WriteString("</li></ul>")
	}

	return buf.String()
}

// firstHeading is the text of the document's first heading, or "".
func firstHeading(doc ast.Node) string {
	text := ""
	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		if heading, ok := node.(*ast.Heading); ok && entering {
			text = extractText(heading)
			return ast.Terminate
		}
		return ast.GoToNext
	})
	return text
}

// extractText gets the plain text content of an AST node.
func extractText(node ast.Node) string {
	var buf bytes.Buffer
	ast.WalkFunc(node, func(n ast.Node, entering bool) ast.WalkStatus {
		if leaf, ok := n.(*ast.Text); ok {
			buf.Write(leaf.Literal)
		}
		return ast.GoToNext
	})
	return buf.String()
}
