package internal

import "text/template"

type pkgHeaderContext struct {
	GeneratorName    string
	GeneratorVersion string
	PackageName      string
	NeedSort         bool
	NeedStrconv      bool
}

type typeStartContext struct {
	TypeName  string
	NeedsSort bool
}

var templates = template.Must(template.New("root").Parse(`
{{ define "package_header" -}}
// Code generated by {{.GeneratorName}} {{.GeneratorVersion}} DO NOT EDIT.
//+build !generate

package {{.PackageName}}

import (
	"github.com/predakanga/bencode_gen/pkg"{{ if .NeedSort }}
	"sort"{{ end }}{{ if .NeedStrconv }}
	"strconv"{{ end }}
)
{{- end }}

{{ define "type_start" }}

func (x *{{.TypeName}}) WriteTo(w pkg.Writer) (err error) { {{- if .NeedsSort }}
	var mapKeys sort.StringSlice{{ end }}
{{- end }}

{{ define "type_end" }}

	return
}
{{- end }}

{{ define "const_byte" }}
	if err = w.WriteByte('{{.}}'); err != nil {
		return
	}
{{- end }}

{{ define "const_string" }}
	if _, err = w.WriteString("{{.}}"); err != nil {
		return
	}
{{- end }}

{{ define "native" }}
	if err = {{.}}.WriteTo(w); err != nil {
		return
	}
{{- end }}

{{ define "bool" }}
	if {{.}} {
		err = w.WriteByte('1') 
	} else {
		err = w.WriteByte('0')
	}
	if err {
		return
	}
{{- end }}

{{ define "int" }}
	if _, err = w.WriteString(strconv.FormatInt(int64({{.}}), 10)); err != nil {
		return
	}
{{- end }}

{{ define "string" }}
	if _, err = w.WriteString(strconv.Itoa(len({{.}}))); err != nil {
		return
	}
	if err = w.WriteByte(':'); err != nil {
		return
	}
	if _, err = w.WriteString({{.}}); err != nil {
		return
	}
{{- end }}

{{ define "map_start" }}
	mapKeys = nil
	for k, _ := range {{.}} {
		mapKeys = append(mapKeys, k)
	}
	sort.Sort(mapKeys)
	for _, k := range mapKeys {
{{- end }}

{{ define "map_end" }}
	}
{{- end }}

{{ define "omitempty_false" }}
	if {{.}} {
{{- end }}

{{ define "omitempty_len" }}
	if len({{.}}) != 0 {
{{- end }}

{{ define "omitempty_nil" }}
	if {{.}} != nil {
{{- end }}

{{ define "omitempty_zero" }}
	if {{.}} != 0 {
{{- end }}

{{ define "omitempty_end" }}
	}
{{- end }}
`))
