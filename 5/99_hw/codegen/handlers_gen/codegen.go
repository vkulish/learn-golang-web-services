package main

// код писать тут

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
	"encoding/json"
)

type apiStruct struct {
	URL  string `json:"url"`
	Auth bool   `json:"auth"`
	Method string `json:"method"`
	HandlerName string
}

// Processes a generic declaration node
func processGenDecl(g *ast.GenDecl, structDecls *map[string]*ast.StructType) {
	for _, spec := range g.Specs {
		currType, ok := spec.(*ast.TypeSpec)
		if !ok {
			//fmt.Printf("SKIP %#T is not ast.TypeSpec\n", spec)
			continue
		}

		var typeName = currType.Name.Name

		currStruct, ok := currType.Type.(*ast.StructType)
		if ok {
			// just save struct type info to use it later
			(*structDecls)[typeName] = currStruct
			continue
		}

		//if typeName == "ApiError" {
		//	fmt.Printf("SKIP known struct %s\n", currType.Name.Name)
		//	continue
		//}

		/*
		if g.Doc == nil {
			fmt.Printf("SKIP struct %#v doesnt have comments\n", currType.Name.Name)
			continue
		}

		needCodegen := false
		for _, comment := range g.Doc.List {
			needCodegen = needCodegen || strings.HasPrefix(comment.Text, "// apigen:api")
		}
		if !needCodegen {
			fmt.Printf("SKIP struct %#v as it doesnt have apigen mark\n", currType.Name.Name)
			continue
		}
		*/
	}
}

func processFuncDecl(f *ast.FuncDecl, 
	                 structDecls *map[string]*ast.StructType,
					 routes *map[string][]apiStruct, // object -> routes
					 out *os.File) {

	var name = f.Name.Name
	var genRule string
	if f.Doc != nil {
		for _, comment := range f.Doc.List {
			if strings.HasPrefix(comment.Text, "// apigen:api") {
				genRule = comment.Text
				break
			}
		}
	}	
	if len(genRule) == 0 {
		return
	}

	genRuleJson := strings.TrimPrefix(genRule, "// apigen:api ")

	spec := &apiStruct{}
	res := json.Unmarshal([]byte(genRuleJson), spec)
	if res != nil {
		fmt.Printf("ERROR: Got wrong json for gen mark: %s\n", genRuleJson)
	}

	fmt.Printf("Generating handler for function %s\n", name)
	
	var nameBuilder strings.Builder
	fmt.Fprintf(&nameBuilder, "handler%s", f.Name.Name)
	spec.HandlerName = nameBuilder.String()

	fmt.Fprint(out, `func `)

	if f.Recv != nil {
		fmt.Fprint(out, `(h `)
		for _, item := range f.Recv.List {
			star, ok := item.Type.(*ast.StarExpr)
			if ok {
				var objectNameBuilder strings.Builder
				fmt.Fprintf(&objectNameBuilder, "%s", star.X)
				fmt.Fprintf(out, " *%s", star.X)
				(*routes)[objectNameBuilder.String()] = append((*routes)[objectNameBuilder.String()], *spec)
			}
			//fmt.Printf("recv types: %T data: %+v\n", item.Type, item.Type)
			//fmt.Fprintf(out, "%s", item.Names[0].Name)
		}
		fmt.Fprint(out, `) `)
	}

	// function args
	fmt.Fprintf(out, "%s(w http.ResponseWriter, r *http.Request) {\n", nameBuilder.String())

	// function body
	var args = make([]string, 0)
	if f.Type != nil {
		for i, item := range f.Type.Params.List {
			fmt.Printf("type: %T data: %+v\n", item, item)
			fmt.Printf("type: %T data: %+v\n", item.Type, item.Type)
			//fmt.Fprintf(out, "%s ", item.Names[0].Name)
			switch item.Type.(type) {
				case *ast.SelectorExpr:
					selector := item.Type.(*ast.SelectorExpr)
					if selector.Sel.Name == "Context" {
						fmt.Fprintln(out, "\tctx := r.Context()")
						args = append(args, "ctx")
					}
				//	fmt.Fprintf(out, "%s.%s", selector.X, selector.Sel.Name)
				case *ast.Ident:
					ident := item.Type.(*ast.Ident)
					fmt.Fprintf(out, "\tvar arg%d %s\n", i, ident.Name)
					args = append(args, fmt.Sprintf("arg%d", i))
				default:
					continue
			}
			
		}
	}

	// call wrapped function
	fmt.Fprintf(out, "\th.%s(", f.Name.Name)
	for i, arg := range args {
		fmt.Fprint(out, arg)
		if i+1 < len(args) {
			fmt.Fprintf(out, ", ")
		}
	}
	fmt.Fprintln(out, ")")

	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)

	// function args
	/*fmt.Fprintf(out, `%s(`, nameBuilder.String())
	if f.Type != nil {
		for i, item := range f.Type.Params.List {
			fmt.Printf("type: %T data: %+v\n", item, item)
			fmt.Printf("type: %T data: %+v\n", item.Type, item.Type)
			fmt.Fprintf(out, "%s ", item.Names[0].Name)
			switch item.Type.(type) {
				case *ast.SelectorExpr:
					selector := item.Type.(*ast.SelectorExpr)
					fmt.Fprintf(out, "%s.%s", selector.X, selector.Sel.Name)
				case *ast.Ident:
					ident := item.Type.(*ast.Ident)
					fmt.Fprintf(out, "%s", ident.Name)
				default:
					continue
			}
			
			if i+1 < len(f.Type.Params.List) {
				fmt.Fprintf(out, ", ")
			}
		}
	}
	fmt.Fprint(out, `) `)
	fmt.Fprintln(out)
	*/
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("wrong number of arguments: got %d but expected %d", len(os.Args)-1, 2)
		return
	}

	// simple validation that input and output files are different
	if os.Args[1] == os.Args[2] {
		log.Fatal("input and output file paths must be different")
		return
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
		return
	}

	out, _ := os.Create(os.Args[2])

	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out) // empty line
	fmt.Fprintln(out, `import "net/http"`)
	fmt.Fprintln(out, `import "context"`)
	fmt.Fprintln(out) // empty line

	// structs map
	// "name" -> "*ast.StructType"
	var structDecls = make(map[string]*ast.StructType)

	// collect information about structures
	for _, f := range node.Decls {
		g, ok := f.(*ast.GenDecl)
		if !ok {
			continue
		}
		processGenDecl(g, &structDecls)
	}

	// process functions
	// routing: maps class "name" -> "handlers"
	routes := make(map[string][]apiStruct, 0)
	for _, f := range node.Decls {
		f, ok := f.(*ast.FuncDecl)
		if !ok {
			continue
		}
		processFuncDecl(f, &structDecls, &routes, out)
	}

	for name, routes := range routes {
		fmt.Printf("create ServeHTTP() for class %s\n", name)
		fmt.Fprintf(out, "func (h *%s) ServeHTTP(w http.ResponseWriter, r *http.Request) {\n", name)
		fmt.Fprintln(out, "\tswitch r.URL.Path {")
		for _, route := range routes {
			fmt.Fprintf(out, "\tcase \"%s\":\n", route.URL)
			fmt.Fprintf(out, "\t\th.%s(w, r)\n", route.HandlerName)
		}
		fmt.Fprintln(out, "\tdefault:")
		fmt.Fprintln(out, "\t\tw.WriteHeader(http.StatusNotFound)")
		fmt.Fprintln(out, "\t}")
		fmt.Fprintln(out, "}")
		fmt.Fprintln(out)
	}
}
