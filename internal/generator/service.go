package generator

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

func Generate(root string) error {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
		Dir:  root,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return err
	}

	buildGlobalRegistry(pkgs)

	for _, pkg := range pkgs {
		for i, file := range pkg.Syntax {
			filename := pkg.GoFiles[i]
			if strings.HasSuffix(filename, "_gen.go") {
				continue
			}

			var structsToGen []structInfo

			ast.Inspect(file, func(n ast.Node) bool {
				genDecl, ok := n.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					return true
				}

				// Check for //gimpl: comment
				impls := parseImpls(genDecl.Doc)

				if len(impls) == 0 {
					return true
				}

				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}

					structsToGen = append(structsToGen, parseStruct(typeSpec.Name.Name, structType, impls, pkg))
				}
				return true
			})

			if len(structsToGen) > 0 {
				if err := generateFile(filename, pkg.Name, structsToGen); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
