package tmpl

import (
	"html/template"
)

// Jankily avoid having to package templates separately from the binary. Only
// doing this because there is only one template.
//go:generate ./gen.sh

func mustParse(name, str string) *template.Template {
	t, err := template.New(name).Parse(str)
	if err != nil {
		panic(err)
	}
	return t
}
