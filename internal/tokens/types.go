package tokens

import (
	"github.com/dave/jennifer/jen"
	"go/types"
)

type CodeToken interface {
	GenerateAST(g *jen.Group)
}

type Container interface {
	SetContents([]CodeToken)
	Contents() []CodeToken
}

type leafToken struct{
	Data string
}

type MapSort struct{}
type Const leafToken
type Int leafToken
type Bool leafToken
type String leafToken
type Native leafToken

type List struct{
	Selector string
	Children []CodeToken
}
type Map struct{
	Selector string
	Cast     *types.TypeName
	Children []CodeToken
}
type OmitEmpty struct{
	Selector    string
	EmptyMethod string
	Children    []CodeToken
}