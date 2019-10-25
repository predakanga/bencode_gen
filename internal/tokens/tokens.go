package tokens

import (
	"github.com/dave/jennifer/jen"
)

/*
	if err = w.WriteByte('{{.}}'); err != nil {
		return
	}
*/
func (c *Const) GenerateAST(g *jen.Group) {
	g.IfFunc(func(group *jen.Group) {
		if len(c.Data) == 1 {
			group.Err().Op("=").Id("w").Dot("WriteByte").Call(jen.LitRune(rune(c.Data[0])))
		} else {
			group.List(jen.Id("_"), jen.Err()).Op("=").Id("w").Dot("WriteString").Call(jen.Lit(c.Data))
		}
		group.Err().Op("!=").Nil()
	}).Block(
		jen.Return(),
	)
}

/*
	if _, err = w.WriteString(strconv.FormatInt(int64({{.}}), 10)); err != nil {
		return
	}
*/
func (tok *Int) GenerateAST(g *jen.Group) {
	g.If(
		jen.List(jen.Id("_"), jen.Err()).Op("=").Id("w").Dot("WriteString").Call(
			jen.Qual("strconv", "FormatInt").Call(jen.Int64().Parens(jen.Id(tok.Data)), jen.Lit(10)),
		),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(),
	)
}

/*
	if {{.}} {
		err = w.WriteByte('1')
	} else {
		err = w.WriteByte('0')
	}
	if err {
		return
	}
*/
func (tok *Bool) GenerateAST(g *jen.Group) {
	g.If(jen.Id(tok.Data)).Block(
		jen.Err().Op("=").Id("w").Dot("WriteByte").Call(jen.LitRune('1')),
	).Else().Block(
		jen.Err().Op("=").Id("w").Dot("WriteByte").Call(jen.LitRune('0')),
	)
	g.If(jen.Err().Op("!=").Nil()).Block(jen.Return())
}

/*
	if _, err = w.WriteString(strconv.Itoa(len({{.}}))); err != nil {
		return
	}
	if err = w.WriteByte(':'); err != nil {
		return
	}
	if _, err = w.WriteString({{.}}); err != nil {
		return
	}
*/
func (tok *String) GenerateAST(g *jen.Group) {
	g.If(
		jen.List(jen.Id("_"), jen.Err()).Op("=").Id("w").Dot("WriteString").Call(
			jen.Qual("strconv", "Itoa").Call(jen.Len(jen.Id(tok.Data))),
		),
		jen.Err().Op("!=").Nil(),
	).Block(jen.Return())
	g.If(
		jen.Err().Op("=").Id("w").Dot("WriteByte").Call(jen.LitRune(':')),
		jen.Err().Op("!=").Nil(),
	).Block(jen.Return())
	g.If(
		jen.List(jen.Id("_"), jen.Err()).Op("=").Id("w").Dot("WriteString").Call(jen.Id(tok.Data)),
		jen.Err().Op("!=").Nil(),
	).Block(jen.Return())
}

/*
	if err = {{.}}.WriteTo(w); err != nil {
		return
	}
*/
func (tok *Native) GenerateAST(g *jen.Group) {
	g.If(
		jen.Err().Op("=").Id(tok.Data).Dot("WriteTo").Call(jen.Id("w")),
		jen.Err().Op("!=").Nil(),
	).Block(jen.Return())
}

func (tok *List) GenerateAST(g *jen.Group) {
	g.For(jen.Id("i").Op(":=").Range().Id(tok.Selector)).BlockFunc(func(sg *jen.Group) {
		for _, child := range tok.Children {
			child.GenerateAST(sg)
		}
	})
}

func (tok *List) Contents() []CodeToken {
	return tok.Children
}

func (tok *List) SetContents(children []CodeToken) {
	tok.Children = children
}

/*
	mapKeys = nil
	for k, _ := range {{.Selector}} {
		mapKeys = append(mapKeys, {{ if (.Import) ne "" }}string(k){{ else }}k{{ end }})
	}
	sort.Sort(mapKeys)
	for _, idx := range mapKeys {
	{{- if (.Import) ne "" }}
		k := {{ .Import }}.{{ .Type }}(idx)
	{{- else }}
		k := idx
	{{- end }}
 */
func (tok *Map) GenerateAST(g *jen.Group) {
	// First, sort the map
	g.Id("mapKeys").Op("=").Nil()
	g.For(
		jen.List(jen.Id("k"), jen.Id("_")).Op(":=").Range().Id(tok.Selector),
	).Block(
		jen.Id("mapKeys").Op("=").AppendFunc(func(sg *jen.Group) {
			sg.Id("mapKeys")
			if tok.Cast != nil {
				sg.String().Parens(jen.Id("k"))
			} else {
				sg.Id("k")
			}
		}),
	)
	g.Qual("sort", "Sort").Call(jen.Id("mapKeys"))
	g.For(
		jen.List(jen.Id("_"), jen.Id("idx")).Op(":=").Range().Id("mapKeys"),
	).BlockFunc(func(sg *jen.Group) {
		if tok.Cast != nil {
			sg.Id("k").Op(":=").Qual(tok.Cast.Pkg().Path(), tok.Cast.Name()).Parens(jen.Id("idx"))
		} else {
			sg.Id("k").Op(":=").Id("idx")
		}
		for _, child := range tok.Children {
			child.GenerateAST(sg)
		}
	})
}

func (tok *Map) Contents() []CodeToken {
	return tok.Children
}

func (tok *Map) SetContents(children []CodeToken) {
	tok.Children = children
}

func (tok *OmitEmpty) GenerateAST(g *jen.Group) {
	g.IfFunc(func(cond *jen.Group) {
		switch tok.EmptyMethod {
		case "len":
			cond.Len(jen.Id(tok.Selector)).Op("!=").Lit(0)
		case "zero":
			cond.Id(tok.Selector).Op("!=").Lit(0)
		case "nil":
			cond.Id(tok.Selector).Op("!=").Nil()
		case "false":
			cond.Id(tok.Selector).Op("!=").False()
		}
	}).BlockFunc(func(sg *jen.Group) {
		for _, child := range tok.Children {
			child.GenerateAST(sg)
		}
	})
}

func (tok *OmitEmpty) Contents() []CodeToken {
	return tok.Children
}

func (tok *OmitEmpty) SetContents(children []CodeToken) {
	tok.Children = children
}

func (tok *MapSort) GenerateAST(g *jen.Group) {
	g.Var().Id("mapKeys").Qual("sort", "StringSlice").Line()
}