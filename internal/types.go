package internal

import (
	"github.com/fatih/structtag"
	"go/ast"
	"go/types"
	"regexp"
	"strings"
)

var splitterRegex = regexp.MustCompile(`([a-z])([A-Z])`)

type OutputMode int

const (
	Normal OutputMode = iota
	Overwrite
	DryRun
)

type FieldInfo struct {
	Name  string
	Field *types.Var
	Tag   *structtag.Tag
}

func (f *FieldInfo) OutputName() string {
	if f.Tag != nil {
		return f.Tag.Name
	}
	return strings.ToLower(splitterRegex.ReplaceAllString(f.Field.Name(), "$1 $2"))
}

type FieldSlice []FieldInfo

func (f FieldSlice) Len() int {
	return len(f)
}

func (f FieldSlice) Less(i, j int) bool {
	return f[i].Name < f[j].Name
}

func (f FieldSlice) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

type Definition struct {
	Ident  *ast.Ident
	Object types.Object
}

type DefinitionSlice []Definition

func (d DefinitionSlice) Len() int {
	return len(d)
}

func (d DefinitionSlice) Less(i, j int) bool {
	return d[i].Ident.Name < d[j].Ident.Name
}

func (d DefinitionSlice) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

type token struct {
	Type string
	Data interface{}
}

type castData struct {
	Selector string
	Import string
	Type string
}
