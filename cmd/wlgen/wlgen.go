package main

import (
	"bytes"
	"embed"
	"encoding/xml"
	"flag"
	"go/format"
	"log"
	"os"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"deedles.dev/wl/protocol"
)

var (
	//go:embed *.tmpl
	tmplFS embed.FS
	tmpl   = template.Must(template.New("base").Funcs(tmplFuncs).ParseFS(tmplFS, "*.tmpl"))

	tmplFuncs = map[string]any{
		"camel": func(v string) string {
			var buf strings.Builder
			buf.Grow(len(v))
			shift := true
			for _, c := range v {
				if c == '_' {
					shift = true
					continue
				}

				if shift {
					c = unicode.ToUpper(c)
				}
				buf.WriteRune(c)
				shift = false
			}
			return buf.String()
		},
		"snake": func(v string) string {
			var buf strings.Builder
			buf.Grow(len(v))
			for i, c := range v {
				if unicode.IsUpper(c) && (i > 0) {
					buf.WriteRune('_')
				}
				buf.WriteRune(unicode.ToLower(c))
			}
			return buf.String()
		},
		"export": func(v string) string {
			if len(v) == 0 {
				return ""
			}

			c, size := utf8.DecodeRuneInString(v)
			if unicode.IsUpper(c) {
				return v
			}

			var buf strings.Builder
			buf.Grow(len(v))
			buf.WriteRune(unicode.ToUpper(c))
			buf.WriteString(v[size:])
			return buf.String()
		},
		"unexport": func(v string) string {
			if len(v) == 0 {
				return ""
			}

			c, size := utf8.DecodeRuneInString(v)
			if unicode.IsLower(c) {
				return v
			}

			var buf strings.Builder
			buf.Grow(len(v))
			buf.WriteRune(unicode.ToLower(c))
			buf.WriteString(v[size:])
			return buf.String()
		},
		"trimPrefix": func(prefix, v string) string {
			return strings.TrimPrefix(v, prefix)
		},
	}
)

func loadXML(path string) (proto protocol.Protocol, err error) {
	file, err := os.Open(path)
	if err != nil {
		return proto, err
	}
	defer file.Close()

	d := xml.NewDecoder(file)
	err = d.Decode(&proto)
	return proto, err
}

type TemplateContext struct {
	Protocol protocol.Protocol
	Package  string
	Prefix   string
}

func main() {
	xmlfile := flag.String("xml", "", "protocol XML file")
	out := flag.String("out", "", "output file (default <xml file>.go)")
	pkg := flag.String("pkg", "wl", "output package name")
	prefix := flag.String("prefix", "wl_", "interface prefix name to strip")
	flag.Parse()

	if *out == "" {
		*out = *xmlfile + ".go"
	}

	proto, err := loadXML(*xmlfile)
	if err != nil {
		log.Fatalf("load XML: %v", err)
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "main.tmpl", TemplateContext{
		Protocol: proto,
		Package:  *pkg,
		Prefix:   *prefix,
	})
	if err != nil {
		log.Fatalf("execute template: %v", err)
	}

	data, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("format output: %v", err)
	}

	file, err := os.Create(*out)
	if err != nil {
		log.Fatalf("create output file: %v", err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		log.Fatalf("write output: %v", err)
	}
}