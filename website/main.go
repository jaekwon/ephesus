package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

var tmpl *template.Template

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

// pages defines the available pages and their markdown sources.
var pages = []struct {
	Path  string // URL path
	Title string // display title
	File  string // markdown file path (relative to repo root)
}{
	{"/", "Main", "../README.md"},
	{"/usdollar", "U.S. Dollar", "../usdollar/README.md"},
	{"/jesus_and_taxes", "Jesus & Taxes", "../jesus_and_taxes/README.md"},
}

func main() {
	var err error
	tmpl, err = template.ParseFiles("home.html.tmpl")
	if err != nil {
		log.Fatal("failed to parse template: ", err)
	}

	for _, pg := range pages {
		pg := pg
		http.HandleFunc(pg.Path, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != pg.Path {
				http.NotFound(w, r)
				return
			}
			servePage(w, r, pg.Path, pg.Title, pg.File)
		})
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("../images"))))
	// Serve images from subdirectories
	http.Handle("/usdollar/images/", http.StripPrefix("/usdollar/images/", http.FileServer(http.Dir("../usdollar/images"))))
	http.Handle("/jesus_and_taxes/images/", http.StripPrefix("/jesus_and_taxes/images/", http.FileServer(http.Dir("../jesus_and_taxes/images"))))

	port := ":5001"
	fmt.Println("Server is running on port", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func servePage(w http.ResponseWriter, r *http.Request, currentPath, title, mdFile string) {
	md, err := os.ReadFile(mdFile)
	if err != nil {
		http.Error(w, "Could not read markdown file", http.StatusInternalServerError)
		log.Println("error reading:", mdFile, err)
		return
	}

	// Fix image paths relative to the page's URL path
	imagePrefix := currentPath
	if imagePrefix == "/" {
		imagePrefix = ""
	}
	md = fixImagePaths(md, imagePrefix)

	content := renderMarkdown(md)
	toc := buildTOC(md)

	var pageNav []PageNav
	for _, pg := range pages {
		pageNav = append(pageNav, PageNav{
			Path:   pg.Path,
			Title:  pg.Title,
			Active: pg.Path == currentPath,
		})
	}

	data := PageData{
		Title:   title,
		Content: template.HTML(content),
		TOC:     template.HTML(toc),
		Pages:   pageNav,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Println("template error:", err)
	}
}

func renderMarkdown(md []byte) []byte {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)

	opts := html.RendererOptions{
		Flags: html.CommonFlags,
	}
	renderer := html.NewRenderer(opts)

	return markdown.Render(p.Parse(md), renderer)
}

// buildTOC generates an HTML table of contents from markdown headings.
func buildTOC(md []byte) string {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	var buf bytes.Buffer
	prevLevel := 0

	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		if heading, ok := node.(*ast.Heading); ok && entering {
			level := heading.Level

			if prevLevel == 0 {
				buf.WriteString("<ul>")
				prevLevel = level
			} else if level > prevLevel {
				for i := prevLevel; i < level; i++ {
					buf.WriteString("<ul>")
				}
			} else if level < prevLevel {
				for i := level; i < prevLevel; i++ {
					buf.WriteString("</li></ul>")
				}
				buf.WriteString("</li>")
			} else {
				buf.WriteString("</li>")
			}

			id := heading.HeadingID
			text := extractText(heading)
			fmt.Fprintf(&buf, `<li><a href="#%s">%s</a>`, id, text)
			prevLevel = level
		}
		return ast.GoToNext
	})

	for i := 0; i < prevLevel; i++ {
		buf.WriteString("</li></ul>")
	}

	return buf.String()
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

func fixImagePaths(md []byte, prefix string) []byte {
	result := strings.ReplaceAll(string(md), "./images/", prefix+"/images/")
	return []byte(result)
}
