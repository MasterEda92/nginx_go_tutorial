package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

const tutorialFile = "NGINX_SETUP.md"

var md = goldmark.New(goldmark.WithExtensions(extension.GFM))

var page = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="de"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Tutorial</title>
<style>
:root { color-scheme: light dark; }
body { max-width: 50rem; margin: 2rem auto; padding: 0 1rem; font: 16px/1.6 system-ui, sans-serif; }
pre { padding: 1rem; overflow-x: auto; background: rgba(127,127,127,.15); border-radius: 6px; }
code { font-family: ui-monospace, monospace; font-size: .9em; }
:not(pre) > code { background: rgba(127,127,127,.15); padding: .1em .3em; border-radius: 4px; }
table { border-collapse: collapse; }
th, td { border: 1px solid rgba(127,127,127,.5); padding: .3rem .6rem; }
blockquote { margin-left: 0; padding-left: 1rem; border-left: 4px solid rgba(127,127,127,.5); }
</style></head><body>
<p><a href="/">&larr; Hello World</a></p>
{{.}}
</body></html>`))

func helloHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html><html><head><title>Hello</title></head><body><h1>Hello World</h1><p><a href="/tutorial">Zum Tutorial</a></p></body></html>`)
}

// Die Datei wird bei jedem Request neu gelesen: Änderungen sind ohne Neustart sichtbar.
func tutorialHandler(w http.ResponseWriter, r *http.Request) {
	src, err := os.ReadFile(tutorialFile)
	if err != nil {
		log.Println("Tutorial lesen:", err)
		http.Error(w, "Tutorial nicht verfügbar", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		log.Println("Markdown rendern:", err)
		http.Error(w, "Rendern fehlgeschlagen", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page.Execute(w, template.HTML(buf.String())) // HTML stammt aus goldmark (rohes HTML wird dort escaped)
}

func main() {
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/tutorial", tutorialHandler)

	// Nur auf localhost lauschen: von außen erreichbar ist die App nur über nginx.
	addr := "127.0.0.1:8081"
	log.Println("Server läuft auf", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
