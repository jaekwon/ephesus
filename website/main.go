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

type PageData struct {
	Content template.HTML
	TOC     template.HTML
}

func main() {
	var err error
	tmpl, err = template.ParseFiles("home.html.tmpl")
	if err != nil {
		log.Fatal("failed to parse template: ", err)
	}

	http.HandleFunc("/", handleHome)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("../images"))))

	port := ":5001"
	fmt.Println("Server is running on port", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

// readmeDir returns the path to look for the README.
// It checks for ../README.md (running from website/ inside the repo).
func readmePath() string {
	if _, err := os.Stat("../README.md"); err == nil {
		return "../README.md"
	}
	return "README.md"
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	md, err := os.ReadFile(readmePath())
	if err != nil {
		http.Error(w, "Could not read README.md", http.StatusInternalServerError)
		log.Println("error reading README:", err)
		return
	}

	// Fix image paths: ./images/ -> /static/images/
	md = fixImagePaths(md)

	content := renderMarkdown(md)
	toc := buildTOC(md)

	data := PageData{
		Content: template.HTML(content),
		TOC:     template.HTML(toc),
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

func fixImagePaths(md []byte) []byte {
	result := strings.ReplaceAll(string(md), "./images/", "/images/")
	return []byte(result)
}
