package main

import (
	"html/template"
)

var tmpl = make(map[string]*template.Template)

func init() {
	m := template.Must
	p := template.ParseFiles
	tmpl["index"] = m(p("templates/index.gohtml", "templates/layout.gohtml"))
	tmpl["create"] = m(p("templates/create.gohtml"))
	tmpl["access"] = m(p("templates/event.gohtml", "templates/layout.gohtml"))
}