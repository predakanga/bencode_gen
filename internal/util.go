package internal

import (
	"github.com/fatih/structtag"
	log "github.com/sirupsen/logrus"
	"go/ast"
	"go/types"
	"io"
	"os"
	"sort"
	"strings"
)

func must(f func() error) {
	if err := f(); err != nil {
		log.Fatal(err)
	}
}

func strContains(haystack []string, needle string) bool {
	i := sort.SearchStrings(haystack, needle)
	return i < len(haystack) && haystack[i] == needle
}

func fileExists(path string) bool {
	res, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !res.IsDir()
}

func interestingDef(id *ast.Ident, obj types.Object, typeNames []string) bool {
	if id.Obj == nil || id.Obj.Kind != ast.Typ {
		return false
	}

	if len(typeNames) != 0 {
		return strContains(typeNames, id.Name)
	}

	if struc, ok := obj.Type().Underlying().(*types.Struct); ok {
		return structHasTag(struc, id.Name)
	}

	return false
}

func isString(typ types.Type) bool {
	strTyp, ok := typ.Underlying().(*types.Basic)
	return ok && (strTyp.Info()&types.IsString != 0)
}

func walkStruct(structName string, x *types.Struct, fn func(FieldInfo) bool) {
	for i := 0; i < x.NumFields(); i++ {
		field := x.Field(i)
		fieldName := field.Name()
		var fieldTag *structtag.Tag
		if tags, err := structtag.Parse(x.Tag(i)); err != nil {
			log.Warnf("Failed to parse tag for %v.%v: %v", structName, fieldName, err)
		} else {
			fieldTag, _ = tags.Get("bencode")
		}

		if field.Embedded() {
			if embedded, ok := field.Type().Underlying().(*types.Struct); ok {
				if fieldTag != nil {
					log.Warnf("struct tags on embedded fields are ignored (%v in %v)", fieldName, structName)
				}
				walkStruct(fieldName, embedded, fn)
			} else {
				log.Warnf("Unsupported embedding in %v: %v", structName, fieldName)
			}
		} else {
			if !fn(FieldInfo{fieldName, field, fieldTag}) {
				return
			}
		}
	}
}

func structHasTag(x types.Type, structName string) (found bool) {
	if struc, ok := x.Underlying().(*types.Struct); ok {
		walkStruct(structName, struc, func(f FieldInfo) bool {
			if f.Tag != nil {
				found = true
				return false
			}
			return true
		})
	}

	return
}

func render(tplName string, w io.Writer, ctx interface{}) {
	if err := templates.ExecuteTemplate(w, tplName, ctx); err != nil {
		log.Fatal(err)
	}
}

func packageBaseName(packagePath string) string {
	return packagePath[strings.LastIndexByte(packagePath, '/')+1:]
}