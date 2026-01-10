package cli

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"
)

var GenCmd = &cobra.Command{
	Use:   "gen [path]",
	Short: "Generate gimpl implementations",
	Long:  "Generate gimpl implementations for structs annotated with gimpl tags.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		return generate(path)
	},
}

func generate(root string) error {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:  root,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return err
	}

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
				impls := make(map[string]InterfaceSet)
				if genDecl.Doc != nil {
					for _, comment := range genDecl.Doc.List {
						text := strings.TrimSpace(comment.Text)
						text = strings.TrimPrefix(text, "//")
						text = strings.TrimSpace(text)

						// Check against supported targets
						for _, target := range []string{"pgimpl"} {
							prefix := target + ":"
							if after, ok := strings.CutPrefix(text, prefix); ok {
								content := after

								set := impls[target]
								for part := range strings.SplitSeq(content, ",") {
									switch strings.TrimSpace(part) {
									case "Entity":
										set.IsEntity = true
									case "CreateEntity":
										set.IsCreate = true
									case "UpdateEntity":
										set.IsUpdate = true
									case "SaveEntity":
										set.IsCreate = true
										set.IsUpdate = true
									}
								}
								impls[target] = set
							}
						}
					}
				}

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

					structsToGen = append(structsToGen, parseStruct(typeSpec.Name.Name, structType, impls))
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

type InterfaceSet struct {
	IsEntity bool
	IsCreate bool
	IsUpdate bool
}

type structInfo struct {
	Name   string
	Fields []fieldInfo
	Impls  map[string]InterfaceSet
}

type fieldImplInfo struct {
	ColumnName string
	Identify   bool
	Update     bool
	Create     bool
}

type fieldInfo struct {
	Name      string
	FieldName string
	Sort      bool
	Filter    bool
	Impls     map[string]fieldImplInfo
}

func parseStruct(name string, st *ast.StructType, impls map[string]InterfaceSet) structInfo {
	info := structInfo{
		Name:  name,
		Impls: impls,
	}

	// Calculate effective requirements for tag parsing
	var isEntity, isCreate, isUpdate bool
	for _, set := range impls {
		if set.IsEntity {
			isEntity = true
		}
		if set.IsCreate {
			isCreate = true
		}
		if set.IsUpdate {
			isUpdate = true
		}
	}

	for _, field := range st.Fields.List {
		if field.Tag == nil {
			continue
		}

		tagValue := strings.Trim(field.Tag.Value, "`")
		tag := reflect.StructTag(tagValue)
		gimplTag := tag.Get("gimpl")
		pgimplTag := tag.Get("pgimpl")

		if gimplTag == "" && pgimplTag == "" {
			continue
		}

		fInfo := fieldInfo{
			Impls: make(map[string]fieldImplInfo),
		}
		if len(field.Names) > 0 {
			fInfo.Name = field.Names[0].Name
		} else {
			// Handle embedded fields if they have a name we can derive
			if ident, ok := field.Type.(*ast.Ident); ok {
				fInfo.Name = ident.Name
			} else {
				continue
			}
		}

		// Parse gimpl tag
		if gimplTag != "" {
			for part := range strings.SplitSeq(gimplTag, ";") {
				kv := strings.Split(part, ":")
				key := strings.TrimSpace(kv[0])
				val := ""
				if len(kv) > 1 {
					val = strings.TrimSpace(kv[1])
				}

				switch key {
				case "field":
					fInfo.FieldName = val
				case "sort":
					if isEntity {
						fInfo.Sort = true
					}
				case "filter":
					if isEntity {
						fInfo.Filter = true
					}
				}
			}
		}

		// Parse pgimpl tag
		pgInfo := fieldImplInfo{}
		if pgimplTag != "" {
			for part := range strings.SplitSeq(pgimplTag, ";") {
				kv := strings.Split(part, ":")
				key := strings.TrimSpace(kv[0])
				val := ""
				if len(kv) > 1 {
					val = strings.TrimSpace(kv[1])
				}

				switch key {
				case "column":
					pgInfo.ColumnName = val
				case "identify":
					if isUpdate {
						pgInfo.Identify = true
					}
				case "update":
					if isUpdate {
						pgInfo.Update = true
					}
				case "create":
					if isCreate {
						pgInfo.Create = true
					}
				}
			}
		}

		if fInfo.FieldName == "" {
			continue
		}
		if pgInfo.ColumnName == "" {
			pgInfo.ColumnName = fInfo.FieldName
		}
		fInfo.Impls["pgimpl"] = pgInfo

		info.Fields = append(info.Fields, fInfo)
	}
	return info
}

func generateFile(originalFile, pkgName string, structs []structInfo) error {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "// Code generated by gimpl. DO NOT EDIT.\n")
	fmt.Fprintf(buf, "package %s\n\n", pkgName)

	for _, s := range structs {
		generateCommonMethods(buf, s)

		// Generate for specific targets
		for target := range s.Impls {
			switch target {
			case "pgimpl":
				generatePgMethods(buf, s)
			}
		}
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// If formatting fails, write unformatted code for debugging
		// But usually we return error
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	outFile := strings.TrimSuffix(originalFile, ".go") + "_gen.go"
	return os.WriteFile(outFile, formatted, 0644)
}

func generateCommonMethods(buf *bytes.Buffer, s structInfo) {
	// Check if any impl requires Entity
	var isEntity bool
	for _, set := range s.Impls {
		if set.IsEntity {
			isEntity = true
			break
		}
	}

	if isEntity {
		// SortPtr
		fmt.Fprintf(buf, "func (e %s) SortPtr(field string) any {\n", s.Name)
		fmt.Fprintf(buf, "\tswitch field {\n")
		for _, f := range s.Fields {
			if f.Sort {
				fmt.Fprintf(buf, "\tcase \"%s\":\n", f.FieldName)
				fmt.Fprintf(buf, "\t\treturn &e.%s\n", f.Name)
			}
		}
		fmt.Fprintf(buf, "\t}\n\treturn nil\n}\n\n")

		// FilterPtr
		fmt.Fprintf(buf, "func (e %s) FilterPtr(field string) any {\n", s.Name)
		fmt.Fprintf(buf, "\tswitch field {\n")
		for _, f := range s.Fields {
			if f.Filter {
				fmt.Fprintf(buf, "\tcase \"%s\":\n", f.FieldName)
				fmt.Fprintf(buf, "\t\treturn &e.%s\n", f.Name)
			}
		}
		fmt.Fprintf(buf, "\t}\n\treturn nil\n}\n\n")
	}
}

func generatePgMethods(buf *bytes.Buffer, s structInfo) {
	set := s.Impls["pgimpl"]

	if set.IsEntity {
		// PgColumn
		fmt.Fprintf(buf, "func (e %s) PgColumn(field string) string {\n", s.Name)
		fmt.Fprintf(buf, "\tswitch field {\n")
		for _, f := range s.Fields {
			pgInfo := f.Impls["pgimpl"]
			fmt.Fprintf(buf, "\tcase \"%s\":\n", f.FieldName)
			fmt.Fprintf(buf, "\t\treturn \"%s\"\n", pgInfo.ColumnName)
		}
		fmt.Fprintf(buf, "\t}\n\treturn \"\"\n}\n\n")

		// PgColumns
		fmt.Fprintf(buf, "func (e %s) PgColumns() []string {\n", s.Name)
		fmt.Fprintf(buf, "\treturn []string{\n")
		for _, f := range s.Fields {
			pgInfo := f.Impls["pgimpl"]
			fmt.Fprintf(buf, "\t\t\"%s\",\n", pgInfo.ColumnName)
		}
		fmt.Fprintf(buf, "\t}\n}\n\n")

		// NewWithPgColumnPtrs
		fmt.Fprintf(buf, "func (e %s) NewWithPgColumnPtrs() (any, []any) {\n", s.Name)
		fmt.Fprintf(buf, "\tn := &%s{}\n", s.Name)
		fmt.Fprintf(buf, "\treturn n, []any{\n")
		for _, f := range s.Fields {
			fmt.Fprintf(buf, "\t\t&n.%s,\n", f.Name)
		}
		fmt.Fprintf(buf, "\t}\n}\n\n")
	}

	if set.IsCreate {
		// CreatePgColumns
		fmt.Fprintf(buf, "func (e %s) CreatePgColumns() []string {\n", s.Name)
		fmt.Fprintf(buf, "\treturn []string{\n")
		for _, f := range s.Fields {
			pgInfo := f.Impls["pgimpl"]
			if pgInfo.Create {
				fmt.Fprintf(buf, "\t\t\"%s\",\n", pgInfo.ColumnName)
			}
		}
		fmt.Fprintf(buf, "\t}\n}\n\n")

		// CreatePgColumnVals
		fmt.Fprintf(buf, "func (e %s) CreatePgColumnVals() []any {\n", s.Name)
		fmt.Fprintf(buf, "\treturn []any{\n")
		for _, f := range s.Fields {
			pgInfo := f.Impls["pgimpl"]
			if pgInfo.Create {
				fmt.Fprintf(buf, "\t\te.%s,\n", f.Name)
			}
		}
		fmt.Fprintf(buf, "\t}\n}\n\n")
	}

	if set.IsUpdate {
		// IdentifyPgColumns
		fmt.Fprintf(buf, "func (e %s) IdentifyPgColumns() []string {\n", s.Name)
		fmt.Fprintf(buf, "\treturn []string{\n")
		for _, f := range s.Fields {
			pgInfo := f.Impls["pgimpl"]
			if pgInfo.Identify {
				fmt.Fprintf(buf, "\t\t\"%s\",\n", pgInfo.ColumnName)
			}
		}
		fmt.Fprintf(buf, "\t}\n}\n\n")

		// IdentifyPgColumnVals
		fmt.Fprintf(buf, "func (e %s) IdentifyPgColumnVals() []any {\n", s.Name)
		fmt.Fprintf(buf, "\treturn []any{\n")
		for _, f := range s.Fields {
			pgInfo := f.Impls["pgimpl"]
			if pgInfo.Identify {
				fmt.Fprintf(buf, "\t\te.%s,\n", f.Name)
			}
		}
		fmt.Fprintf(buf, "\t}\n}\n\n")

		// UpdatePgColumns
		fmt.Fprintf(buf, "func (e %s) UpdatePgColumns() []string {\n", s.Name)
		fmt.Fprintf(buf, "\treturn []string{\n")
		for _, f := range s.Fields {
			pgInfo := f.Impls["pgimpl"]
			if pgInfo.Update {
				fmt.Fprintf(buf, "\t\t\"%s\",\n", pgInfo.ColumnName)
			}
		}
		fmt.Fprintf(buf, "\t}\n}\n\n")

		// UpdatePgColumnVals
		fmt.Fprintf(buf, "func (e %s) UpdatePgColumnVals() []any {\n", s.Name)
		fmt.Fprintf(buf, "\treturn []any{\n")
		for _, f := range s.Fields {
			pgInfo := f.Impls["pgimpl"]
			if pgInfo.Update {
				fmt.Fprintf(buf, "\t\te.%s,\n", f.Name)
			}
		}
		fmt.Fprintf(buf, "\t}\n}\n\n")
	}
}
