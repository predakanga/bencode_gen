package internal

import (
	"fmt"
	. "github.com/predakanga/bencode_gen/internal/tokens"
	log "github.com/sirupsen/logrus"
	"go/types"
	"sort"
)

func (pg *PackageGenerator) typeTokens(selector string, typ types.Type, ctx *typeContext) []CodeToken {
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
			return pg.structTokens(selector, castType, ctx)
		case *types.Map:
			return pg.mapTokens(selector, castType, ctx)
		case *types.Slice:
			return pg.listTokens(selector, castType.Elem(), ctx)
		case *types.Array:
			return pg.listTokens(selector, castType.Elem(), ctx)
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

func (pg *PackageGenerator) boolTokens(selector string) []CodeToken {
	return []CodeToken{
		&Const{"i"},
		&Bool{selector},
		&Const{"e"},
	}
}

func (pg *PackageGenerator) intTokens(selector string) []CodeToken {
	return []CodeToken{
		&Const{"i"},
		&Int{selector},
		&Const{"e"},
	}
}

func (pg *PackageGenerator) stringTokens(selector string) []CodeToken {
	return []CodeToken{&String{selector}}
}

func (pg *PackageGenerator) nativeTokens(selector string) []CodeToken {
	return []CodeToken{&Native{selector}}
}

func (pg *PackageGenerator) listTokens(selector string, elemType types.Type, ctx *typeContext) []CodeToken {
	return []CodeToken{
		&Const{"l"},
		&List{
			selector,
			pg.typeTokens("i", elemType, ctx),
		},
		&Const{"e"},
	}
}

func (pg *PackageGenerator) mapTokens(selector string, typ *types.Map, ctx *typeContext) []CodeToken {
	var castTo *types.TypeName
	// TODO: Possibly extend this to support go.Stringer
	keyType := typ.Key()
	valType := typ.Elem()
	if !isString(keyType) {
		log.Fatalf("Can't encode %v: map keys may only be strings (not %v)", selector, keyType)
	}
	// Tell the type generator that we'll need the mapKeys var
	ctx.NeedsSort = true

	if namedType, ok := keyType.(*types.Named); ok {
		castTo = namedType.Obj()
	} else if _, ok := keyType.(*types.Basic); !ok {
		log.Fatalf("Can't encode %v: expected types.Named or types.Basic, got %T", selector, keyType)
	}

	childTokens := pg.typeTokens("idx", keyType, ctx)
	childTokens = append(childTokens, pg.typeTokens(selector+"[k]", valType, ctx)...)

	return []CodeToken{
		&Const{"d"},
		&Map{
			selector,
			castTo,
			childTokens,
		},
		&Const{"e"},
	}
}

func (pg *PackageGenerator) structTokens(selector string, typ *types.Struct, ctx *typeContext) (toRet []CodeToken) {
	// Dict keys must be sorted, so fetch the fields and sort them
	var fields FieldSlice
	walkStruct(typ.String(), typ, func(f FieldInfo) bool {
		fields = append(fields, f)
		return true
	})
	sort.Sort(fields)

	toRet = []CodeToken{&Const{"d"}}
	for _, f := range fields {
		var fieldTokens []CodeToken

		// First output the (const) field name
		outputName := f.OutputName()
		fieldTokens = append(fieldTokens, &Const{fmt.Sprintf("%d:%s", len(outputName), outputName)})

		// Then encode the value
		fieldSelector := selector + "." + f.Name
		fieldTokens = append(fieldTokens, pg.typeTokens(fieldSelector, f.Field.Type(), ctx)...)

		// Wrap it in an omit-empty token if need be and store it
		if f.Tag != nil && f.Tag.HasOption("omitempty") {
			emptyMethod := emptyMethod(f.Field.Type().Underlying())
			if emptyMethod == "" {
				panic(fmt.Errorf("omitempty is not supported by type %v (field %v)", f.Field.Type(), fieldSelector))
			}
			toRet = append(toRet, &OmitEmpty{fieldSelector, emptyMethod, fieldTokens})
		} else {
			toRet = append(toRet, fieldTokens...)
		}
	}
	toRet = append(toRet, &Const{"e"})

	return
}
