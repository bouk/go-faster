package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
)

var modifiedFunctions = make(map[string]bool)

func callToReceiveCall(expr *ast.CallExpr) *ast.UnaryExpr {
	return &ast.UnaryExpr{Op: token.ARROW, X: expr}
}

func typeToChanType(t ast.Expr) *ast.ChanType {
	return &ast.ChanType{Value: t, Dir: ast.SEND | ast.RECV, Arrow: token.NoPos}
}

func increaseSpeed(f *ast.FuncDecl) *ast.FuncDecl {
	returnType := f.Type.Results.List[0].Type

	res := &ast.FuncDecl{}
	res.Body = &ast.BlockStmt{}
	res.Doc = f.Doc
	res.Name = f.Name
	res.Recv = f.Recv

	res.Type = &ast.FuncType{
		Params: f.Type.Params,
		Results: &ast.FieldList{
			List: []*ast.Field{
				&ast.Field{
					Type: typeToChanType(returnType),
				},
			},
		},
	}

	resultObject := ast.NewObj(ast.Var, "result")
	resultIdent := &ast.Ident{
		NamePos: token.NoPos,
		Name:    "result",
		Obj:     resultObject,
	}

	innerFunc := &ast.FuncLit{
		Body: f.Body,
		Type: &ast.FuncType{
			Results: f.Type.Results,
		},
	}

	res.Body.List = append(res.Body.List, &ast.AssignStmt{
		Lhs:    []ast.Expr{resultIdent},
		TokPos: token.NoPos,
		Tok:    token.DEFINE,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun:      ast.NewIdent("make"),
				Args:     []ast.Expr{typeToChanType(returnType)},
				Ellipsis: token.NoPos,
			},
		},
	})

	res.Body.List = append(res.Body.List, &ast.GoStmt{
		Call: &ast.CallExpr{
			Fun: &ast.FuncLit{
				Type: &ast.FuncType{Params: &ast.FieldList{}},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.SendStmt{
							Chan:  resultIdent,
							Value: &ast.CallExpr{Ellipsis: token.NoPos, Fun: innerFunc},
						},
					},
				},
			},
		},
	})

	res.Body.List = append(res.Body.List, &ast.ReturnStmt{Results: []ast.Expr{resultIdent}})

	return res
}

type visitor func(ast.Node)

func (v visitor) Visit(node ast.Node) ast.Visitor {
	v(node)
	return v
}

func wrapInReceive(expr ast.Expr) ast.Expr {
	if callExpr, ok := expr.(*ast.CallExpr); ok && callExpr.Lparen != token.NoPos {
		if ident, ok := callExpr.Fun.(*ast.Ident); ok && modifiedFunctions[ident.Name] {
			callExpr.Lparen = token.NoPos
			return callToReceiveCall(callExpr)
		}
	}
	return expr
}

func improveCallExprs(list []ast.Expr) {
	for i, node := range list {
		list[i] = wrapInReceive(node)
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s [inputfile]\n", os.Args[0])
		return
	}
	fset := token.NewFileSet()
	inputFile, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse %s\n", os.Args[1])
		return
	}

	for i, decl := range inputFile.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Recv == nil {
			if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) == 1 {
				modifiedFunctions[funcDecl.Name.Name] = true
				inputFile.Decls[i] = increaseSpeed(funcDecl)
			}
		}
	}

	for _, decl := range inputFile.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			ast.Walk(visitor(func(node ast.Node) {
				switch n := node.(type) {
				case *ast.Field:
					n.Type = wrapInReceive(n.Type)

				case *ast.Ellipsis:
					if n.Elt != nil {
						n.Elt = wrapInReceive(n.Elt)
					}

				case *ast.FuncLit:

				case *ast.CompositeLit:
					if n.Type != nil {
						n.Type = wrapInReceive(n.Type)
					}
					improveCallExprs(n.Elts)

				case *ast.ParenExpr:
					n.X = wrapInReceive(n.X)

				case *ast.SelectorExpr:
					n.X = wrapInReceive(n.X)

				case *ast.IndexExpr:
					n.X = wrapInReceive(n.X)
					n.Index = wrapInReceive(n.Index)

				case *ast.SliceExpr:
					n.X = wrapInReceive(n.X)
					if n.Low != nil {
						n.Low = wrapInReceive(n.Low)
					}
					if n.High != nil {
						n.High = wrapInReceive(n.High)
					}
					if n.Max != nil {
						n.Max = wrapInReceive(n.Max)
					}

				case *ast.TypeAssertExpr:
					n.X = wrapInReceive(n.X)
					if n.Type != nil {
						n.Type = wrapInReceive(n.Type)
					}

				case *ast.CallExpr:
					n.Fun = wrapInReceive(n.Fun)
					improveCallExprs(n.Args)

				case *ast.StarExpr:
					n.X = wrapInReceive(n.X)

				case *ast.UnaryExpr:
					n.X = wrapInReceive(n.X)

				case *ast.BinaryExpr:
					n.X = wrapInReceive(n.X)
					n.Y = wrapInReceive(n.Y)

				case *ast.KeyValueExpr:
					n.Key = wrapInReceive(n.Key)
					n.Value = wrapInReceive(n.Value)

				// Types
				case *ast.ArrayType:
					if n.Len != nil {
						n.Len = wrapInReceive(n.Len)
					}
					n.Elt = wrapInReceive(n.Elt)

				case *ast.MapType:
					n.Key = wrapInReceive(n.Key)
					n.Value = wrapInReceive(n.Value)

				case *ast.ChanType:
					n.Value = wrapInReceive(n.Value)

				case *ast.ExprStmt:
					n.X = wrapInReceive(n.X)

				case *ast.SendStmt:
					n.Chan = wrapInReceive(n.Chan)
					n.Value = wrapInReceive(n.Value)

				case *ast.IncDecStmt:
					n.X = wrapInReceive(n.X)

				case *ast.AssignStmt:
					improveCallExprs(n.Lhs)
					improveCallExprs(n.Rhs)

				case *ast.ReturnStmt:
					improveCallExprs(n.Results)

				case *ast.IfStmt:
					n.Cond = wrapInReceive(n.Cond)

				case *ast.CaseClause:
					improveCallExprs(n.List)

				case *ast.SwitchStmt:
					if n.Tag != nil {
						n.Tag = wrapInReceive(n.Tag)
					}

				case *ast.ForStmt:
					if n.Cond != nil {
						n.Cond = wrapInReceive(n.Cond)
					}

				case *ast.RangeStmt:
					n.Key = wrapInReceive(n.Key)
					if n.Value != nil {
						n.Value = wrapInReceive(n.Value)
					}
					n.X = wrapInReceive(n.X)

				case *ast.ValueSpec:
					if n.Type != nil {
						n.Type = wrapInReceive(n.Type)
					}
					improveCallExprs(n.Values)

				case *ast.TypeSpec:
					n.Type = wrapInReceive(n.Type)

				}
			}), funcDecl)
		}
	}

	format.Node(os.Stdout, fset, inputFile)
}
