package main

// код писать тут

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strings"
)

type apiStruct struct {
	URL  string `json:"url"`
	Auth bool   `json:"auth"`
	Method string `json:"method"`
	HandlerName string
}

type apiValidationRule struct {
	required bool
	min int
	max int
	param_name string
	enum string
	defValue string
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
	}
}

func parseApiValidationRule(tag string) (result *apiValidationRule, ok bool) {
	result = &apiValidationRule{}
	ok = false
	if len(tag) == 0 {
		return
	}

	//	Нам доступны следующие метки валидатора-заполнятора `apivalidator`:
	//	* `required` - поле не должно быть пустым (не должно иметь значение по-умолчанию)
	//	* `paramname` - если указано - то брать из параметра с этим именем, иначе `lowercase` от имени
	//	* `enum` - "одно из"
	//	* `default` - если указано и приходит пустое значение (значение по-умолчанию) - устанавливать то что написано указано в `default`
	//	* `min` - >= X для типа `int`, для строк `len(str)` >=
	//	* `max` - <= X для типа `int`
	
	sentences := strings.Split(tag, ",")
	if len(sentences) == 0 {
		return
	}

	for _, rule := range sentences {
		ruleParts := strings.Split(rule, "=")
		if len(ruleParts) == 0 {
			fmt.Println("ERROR: faced with maflformed apivalidator rule:", rule)
			continue
		}

		ok = true

		switch ruleParts[0] {
		case "required":
			result.required = true
		}
	}

	return result, ok
}

func processFuncDecl(f *ast.FuncDecl, 
	                 structDecls *map[string]*ast.StructType,
					 routes *map[string][]apiStruct, // object -> routes
					 out *os.File) {

	var wrappedFuncName = f.Name.Name

	// getting rules for processing arguments
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
		return
	}

	fmt.Printf("Generating handler for function %s\n", wrappedFuncName)
	
	var handlerNameBuilder strings.Builder
	fmt.Fprintf(&handlerNameBuilder, "handler%s", wrappedFuncName)
	spec.HandlerName = handlerNameBuilder.String()

	fmt.Fprint(out, `func `)

	// if this function is a class method, then we should 
	// determine which one class related to this function
	// and make right signature.
	if f.Recv != nil {
		fmt.Fprint(out, `(h `)
		for _, item := range f.Recv.List {
			star, ok := item.Type.(*ast.StarExpr)
			if ok {
				var objectNameBuilder strings.Builder
				fmt.Fprintf(&objectNameBuilder, "%s", star.X)
				fmt.Fprintf(out, "*%s", star.X)
				fmt.Printf("|- this is a method of %s\n", star.X)
				// store relation between class name and its handler
				(*routes)[objectNameBuilder.String()] = append((*routes)[objectNameBuilder.String()], *spec)
			}
			//fmt.Printf("recv types: %T data: %+v\n", item.Type, item.Type)
			//fmt.Fprintf(out, "%s", item.Names[0].Name)
		}
		fmt.Fprint(out, `) `)
	}

	// function args
	fmt.Fprintf(out, "%s(w http.ResponseWriter, r *http.Request) {\n", handlerNameBuilder.String())

	// function body
	var args = make([]string, 0)
	if f.Type != nil {
		fmt.Println("|- process args:")
		for i, item := range f.Type.Params.List {
			//fmt.Printf("  |- type: %T data: %+v\n", item, item)
			fmt.Printf("  |- type: %T data: %+v\n", item.Type, item.Type)
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
					argName := fmt.Sprintf("arg%d", i)
					argType := ident.Name
					fmt.Fprintf(out, "\tvar %s %s\n", argName, argType) // i.e. "var argN SomeType"
					//fmt.Printf("  |- type: %T data: %+v\n", item, item)

					//filling variable in accordance with its validation rule
					structDecl, ok := (*structDecls)[argType]
					if structDecl.Fields != nil && ok {
						for _, field := range structDecl.Fields.List {
							fieldName := field.Names[0].Name
							//fmt.Printf("%s has field %+v of type %T\n", argType, field, field)
							fmt.Fprintf(out, "\t%s.%s = r.URL.Query().Get(\"%s\")\n", argName, fieldName, fieldName)

							if field.Tag != nil {
								tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1]) // trim ` brackets
								apiValidationStr := tag.Get("apivalidator")
								rule, ok := parseApiValidationRule(apiValidationStr)
								if ok {
									if rule.required {
										fmt.Fprintf(out, "\tif len(%s.%s) == 0 {\n", argName, fieldName)
										fmt.Fprintln(out, "\t\tw.WriteHeader(http.StatusBadRequest)")
										fmt.Fprintln(out, "\t\treturn")
										fmt.Fprintln(out, "\t}")
									}
								}
							}
						}
					}
					args = append(args, argName)
				default:
					continue
			}
			
		}
	}

	// call wrapped function
	fmt.Fprintf(out, "\th.%s(", wrappedFuncName)
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
