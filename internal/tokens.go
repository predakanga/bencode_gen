package internal

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"go/types"
	"sort"
)

func mergeConstTokens(in []token) (out []token) {
	constBuffer := ""
	for len(in) != 0 {
		var tok token
		tok, in = in[0], in[1:]
		if tok.Type == "const" {
			constBuffer = constBuffer + tok.Data
		} else {
			if constBuffer != "" {
				out = append(out, token{"const", constBuffer})
				constBuffer = ""
			}
			out = append(out, tok)
		}
	}
	if constBuffer != "" {
		out = append(out, token{"const", constBuffer})
	}

	return
}

func omitEmptyToken(typ types.Type, selector string) token {
	switch elemType := typ.(type) {
	case *types.Array, *types.Map, *types.Slice:
		return token{"omitempty_len", selector}
	case *types.Interface, *types.Pointer:
		return token{"omitempty_nil", selector}
	case *types.Basic:
		switch {
		case elemType.Info()&types.IsBoolean != 0:
			return token{"omitempty_false", selector}
		case elemType.Info()&types.IsNumeric != 0:
			return token{"omitempty_zero", selector}
		case elemType.Info()&types.IsString != 0:
			return token{"omitempty_len", selector}
		}
	}
	panic(fmt.Errorf("omitempty is not supported by type %v (field %v)", typ, selector))
}

func (pg *PackageGenerator) typeTokens(selector string, typ types.Type) []token {
	// Start at the outer-most type and drill down until we find one we support
	var lastType types.Type
	curType := typ

	for curType != lastType {
		// Special cases
		if types.Implements(curType, pg.bencodeInterface) {
			log.Debugf("Found native support in %v", curType.String())
			return pg.nativeTokens(selector)
		}
		if curType == pg.durationType {
			return pg.intTokens(selector + ".Seconds()")
		}

		// Basic types
		switch castType := curType.(type) {
		case *types.Pointer:
			// Pointers are another special case - for fields, etc, we want to dereference them first
			selector = "*" + selector
			curType = castType.Elem()
			continue
		case *types.Struct:
			return pg.structTokens(selector, castType)
		case *types.Map:
			return pg.mapTokens(selector, castType)
		case *types.Slice:
			return pg.listTokens(selector, castType.Elem())
		case *types.Array:
			return pg.listTokens(selector, castType.Elem())
		case *types.Basic:
			switch {
			case castType.Info()&types.IsBoolean != 0:
				return pg.boolTokens(selector)
			case castType.Info()&types.IsInteger != 0:
				return pg.intTokens(selector)
			case castType.Info()&types.IsString != 0:
				return pg.stringTokens(selector)
			}
		}

		// Dig further
		lastType = curType
		curType = curType.Underlying()
	}

	panic(fmt.Errorf("unsupported type: %v", typ))
}

func (pg *PackageGenerator) boolTokens(selector string) []token {
	return []token{
		{"const", "i"},
		{"bool", selector},
		{"const", "e"},
	}
}

func (pg *PackageGenerator) intTokens(selector string) []token {
	return []token{
		{"const", "i"},
		{"int", selector},
		{"const", "e"},
	}
}

func (pg *PackageGenerator) stringTokens(selector string) []token {
	return []token{{"string", selector}}
}

func (pg *PackageGenerator) nativeTokens(selector string) []token {
	return []token{{"native", selector}}
}

func (pg *PackageGenerator) durationTokens(selector string) []token {
	return []token{
		{"const", "i"},
		{"duration", selector},
		{"const", "e"},
	}
}

func (pg *PackageGenerator) listTokens(selector string, elemType types.Type) []token {
	tokens := []token{
		{"const", "l"},
		{"list_start", selector},
	}
	tokens = append(tokens, pg.typeTokens("i", elemType)...)
	tokens = append(tokens, token{"list_end", selector}, token{"const", "e"})

	return tokens
}

func (pg *PackageGenerator) mapTokens(selector string, typ *types.Map) []token {
	// TODO: Possibly extend this to support go.Stringer
	keyType := typ.Key()
	valType := typ.Elem()
	if !isString(keyType) {
		log.Fatalf("Can't encode %v: map keys may only be strings", selector)
	}

	tokens := []token{{"const", "d"}, {"map_start", selector}}
	tokens = append(tokens, pg.typeTokens("k", keyType)...)
	tokens = append(tokens, pg.typeTokens(selector+"[k]", valType)...)
	tokens = append(tokens, token{"map_end", selector}, token{"const", "e"})

	return tokens
}

func (pg *PackageGenerator) structTokens(selector string, typ *types.Struct) []token {
	// Dict keys must be sorted, so fetch the fields and sort them
	var fields FieldSlice
	walkStruct(typ.String(), typ, func(f FieldInfo) bool {
		fields = append(fields, f)
		return true
	})
	sort.Sort(fields)

	tokens := []token{{"const", "d"}}
	for _, f := range fields {
		fieldSelector := selector + "." + f.Name
		if f.Tag != nil && f.Tag.HasOption("omitempty") {
			tokens = append(tokens, omitEmptyToken(f.Field.Type().Underlying(), fieldSelector))
		}
		// First output the (const) field name
		outputName := f.OutputName()
		tokens = append(tokens, token{"const", fmt.Sprintf("%d:%s", len(outputName), outputName)})
		// Then encode the value
		elemTokens := pg.typeTokens(fieldSelector, f.Field.Type())
		tokens = append(tokens, elemTokens...)
		if f.Tag != nil && f.Tag.HasOption("omitempty") {
			tokens = append(tokens, token{"omitempty_end", fieldSelector})
		}
	}
	tokens = append(tokens, token{"const", "e"})

	return tokens
}
