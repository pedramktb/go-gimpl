package generator

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"reflect"
	"strings"

	"golang.org/x/tools/go/packages"
)

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

	// Delegation support
	IsDelegate         bool
	Prefix             string
	TypeName           string
	DelegateImportPath string
}

type structTypeInfo struct {
	Type    *ast.StructType
	Impls   map[string]InterfaceSet
	PkgPath string
}

var globalStructRegistry = make(map[string]map[string]structTypeInfo)

func buildGlobalRegistry(pkgs []*packages.Package) {
	visited := make(map[string]bool)
	var visit func(pkg *packages.Package)
	visit = func(pkg *packages.Package) {
		if visited[pkg.PkgPath] {
			return
		}
		visited[pkg.PkgPath] = true

		// Process this package
		m := make(map[string]structTypeInfo)
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				genDecl, ok := n.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					return true
				}

				// Parse doc comments for the GenDecl
				impls := parseImpls(genDecl.Doc)

				// Assign to all specs in this block
				for _, spec := range genDecl.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if ok {
						m[ts.Name.Name] = structTypeInfo{
							Type:    st,
							Impls:   impls,
							PkgPath: pkg.PkgPath,
						}
					}
				}
				return true
			})
		}
		globalStructRegistry[pkg.PkgPath] = m

		// Recurse into imports
		for _, imp := range pkg.Imports {
			visit(imp)
		}
	}

	for _, pkg := range pkgs {
		visit(pkg)
	}
}

func parseStruct(name string, st *ast.StructType, impls map[string]InterfaceSet, currentPkg *packages.Package) structInfo {
	info := structInfo{
		Name:  name,
		Impls: impls,
	}
	info.Fields = collectFields(st, impls, currentPkg, "", "")
	return info
}

func collectFields(st *ast.StructType, impls map[string]InterfaceSet, currentPkg *packages.Package, prefix string, columnPrefix string) []fieldInfo {
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

	var fields []fieldInfo

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
		name := ""
		if len(field.Names) > 0 {
			name = field.Names[0].Name
		} else {
			// Handle embedded fields if they have a name we can derive
			if ident, ok := field.Type.(*ast.Ident); ok {
				name = ident.Name
			} else {
				continue
			}
		}

		if prefix != "" {
			fInfo.Name = prefix + "." + name
		} else {
			fInfo.Name = name
		}

		// Parse tags for current field
		currentPgInfo, gimplInfo, gimplTagParams := parseTags(pgimplTag, gimplTag, isCreate, isUpdate, isEntity)

		fInfo.FieldName = gimplInfo.FieldName
		fInfo.Sort = gimplInfo.Sort
		fInfo.Filter = gimplInfo.Filter

		// Handle flattening/delegation
		if gimplTagParams.IsFlat {
			typeName, pkgPath, resolved := resolveFieldType(field, currentPkg)

			if resolved {
				if pkgStructs, ok := globalStructRegistry[pkgPath]; ok {
					if info, ok := pkgStructs[typeName]; ok {
						if shouldDelegate(info) {
							// Delegation
							fInfo.IsDelegate = true
							fInfo.Prefix = columnPrefix + gimplTagParams.Prefix
							fInfo.TypeName = resolveDelegateTypeName(pkgPath, currentPkg.PkgPath, typeName, field)

							if pkgPath != currentPkg.PkgPath {
								fInfo.DelegateImportPath = pkgPath
							}

							fields = append(fields, fInfo)
							continue
						}

						// Inlining
						nestedFields := collectFields(info.Type, impls, currentPkg, fInfo.Name, columnPrefix+gimplTagParams.Prefix)
						fields = append(fields, nestedFields...)
						continue
					}
				}
			}
		}

		if fInfo.FieldName == "" {
			continue
		}
		if currentPgInfo.ColumnName == "" {
			currentPgInfo.ColumnName = fInfo.FieldName
		}

		if columnPrefix != "" {
			fInfo.FieldName = columnPrefix + fInfo.FieldName
			currentPgInfo.ColumnName = columnPrefix + currentPgInfo.ColumnName
		}

		fInfo.Impls["pgimpl"] = currentPgInfo

		fields = append(fields, fInfo)
	}
	return fields
}

// Helper structs for tag parsing results
type gimplTagInfo struct {
	FieldName string
	Sort      bool
	Filter    bool
}

type gimplParams struct {
	IsFlat bool
	Prefix string
}

func parseTags(pgimplTag, gimplTag string, isCreate, isUpdate, isEntity bool) (fieldImplInfo, gimplTagInfo, gimplParams) {
	pgInfo := fieldImplInfo{}
	gInfo := gimplTagInfo{}
	gParams := gimplParams{}

	// Parse pgimpl tag
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
				gInfo.FieldName = val
			case "sort":
				if isEntity {
					gInfo.Sort = true
				}
			case "filter":
				if isEntity {
					gInfo.Filter = true
				}
			case "flat":
				gParams.IsFlat = true
			case "prefix":
				gParams.Prefix = val
			}
		}
	}

	return pgInfo, gInfo, gParams
}

func parseImpls(doc *ast.CommentGroup) map[string]InterfaceSet {
	impls := make(map[string]InterfaceSet)
	if doc != nil {
		for _, comment := range doc.List {
			text := strings.TrimSpace(comment.Text)
			text = strings.TrimPrefix(text, "//")
			text = strings.TrimSpace(text)

			for _, target := range []string{"pgimpl"} {
				prefix := target + ":"
				if after, ok := strings.CutPrefix(text, prefix); ok {
					set := impls[target]
					for part := range strings.SplitSeq(after, ",") {
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
	return impls
}

func resolveFieldType(field *ast.Field, currentPkg *packages.Package) (typeName string, pkgPath string, resolved bool) {
	if ident, ok := field.Type.(*ast.Ident); ok {
		return ident.Name, currentPkg.PkgPath, true
	}

	if star, ok := field.Type.(*ast.StarExpr); ok {
		if ident, ok := star.X.(*ast.Ident); ok {
			return ident.Name, currentPkg.PkgPath, true
		} else if sel, ok := star.X.(*ast.SelectorExpr); ok {
			if xIdent, ok := sel.X.(*ast.Ident); ok {
				return resolveSelector(xIdent.Name, sel.Sel.Name, currentPkg)
			}
		}
	}

	if sel, ok := field.Type.(*ast.SelectorExpr); ok {
		if xIdent, ok := sel.X.(*ast.Ident); ok {
			return resolveSelector(xIdent.Name, sel.Sel.Name, currentPkg)
		}
	}

	return "", "", false
}

func resolveSelector(pkgName, typeName string, currentPkg *packages.Package) (string, string, bool) {
	for _, imp := range currentPkg.Imports {
		if imp.Name == pkgName {
			return typeName, imp.PkgPath, true
		}
		// Fallback for when Name is empty or matches implicit name
		if strings.HasSuffix(imp.PkgPath, pkgName) || strings.HasSuffix(imp.PkgPath, "/"+pkgName) {
			return typeName, imp.PkgPath, true
		}
	}
	return "", "", false
}

func shouldDelegate(info structTypeInfo) bool {
	if set, ok := info.Impls["pgimpl"]; ok {
		return set.IsEntity || set.IsCreate || set.IsUpdate
	}
	return false
}

func resolveDelegateTypeName(pkgPath, currentPkgPath, typeName string, field *ast.Field) string {
	if pkgPath == currentPkgPath {
		return typeName
	}

	var buf bytes.Buffer
	// Trim the * if it's a pointer, for the type assertion check in generator
	// But actually we want the type that is "assertable".
	// The generator does: if val, ok := inner.(*TYPE); ok
	// If the field is *pkg.Type, we want pkg.Type.

	node := field.Type
	if star, ok := field.Type.(*ast.StarExpr); ok {
		node = star.X
	}
	format.Node(&buf, token.NewFileSet(), node)
	return buf.String()
}
