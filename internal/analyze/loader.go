package analyze

import (
	"fmt"
	"go/types"
	"reflect"

	"golang.org/x/tools/go/packages"
)

// LoadMode specifies what information to load from packages.
const LoadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedSyntax |
	packages.NeedTypes |
	packages.NeedTypesInfo |
	packages.NeedImports

// Analyzer loads Go packages and builds a type graph.
type Analyzer struct {
	graph     *TypeGraph
	typeCache map[types.Type]*TypeInfo // Cache to handle recursive types
}

// NewAnalyzer creates a new Analyzer.
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		graph:     NewTypeGraph(),
		typeCache: make(map[types.Type]*TypeInfo),
	}
}

// LoadPackages loads the specified packages and builds the type graph.
// Patterns are standard Go package patterns (e.g., "./store", "caster-generator/warehouse").
func (a *Analyzer) LoadPackages(patterns ...string) (*TypeGraph, error) {
	cfg := &packages.Config{
		Mode: LoadMode,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	// Check for package errors
	var errs []error
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			errs = append(errs, e)
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("package errors: %v", errs)
	}

	// Process each package
	for _, pkg := range pkgs {
		if err := a.processPackage(pkg); err != nil {
			return nil, fmt.Errorf("failed to process package %s: %w", pkg.PkgPath, err)
		}
	}

	return a.graph, nil
}

// Graph returns the current type graph.
func (a *Analyzer) Graph() *TypeGraph {
	return a.graph
}

// processPackage extracts types from a loaded package.
func (a *Analyzer) processPackage(pkg *packages.Package) error {
	pkgInfo := &PackageInfo{
		Path: pkg.PkgPath,
		Name: pkg.Name,
	}

	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)

		// Only process type names (not variables, constants, functions)
		typeName, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}

		// Only process exported types
		if !typeName.Exported() {
			continue
		}

		typeID := TypeID{
			PkgPath: pkg.PkgPath,
			Name:    name,
		}

		typeInfo := a.analyzeType(typeName.Type())
		typeInfo.ID = typeID

		a.graph.Types[typeID] = typeInfo
		pkgInfo.Types = append(pkgInfo.Types, typeID)
	}

	a.graph.Packages[pkg.PkgPath] = pkgInfo
	return nil
}

// analyzeType recursively analyzes a go/types.Type and returns a TypeInfo.
func (a *Analyzer) analyzeType(t types.Type) *TypeInfo {
	// Check cache to handle recursive types
	if cached, ok := a.typeCache[t]; ok {
		return cached
	}

	info := &TypeInfo{
		GoType: t,
	}

	// Pre-cache to handle recursive types (we'll fill in details)
	a.typeCache[t] = info

	switch tt := t.(type) {
	case *types.Named:
		a.analyzeNamedType(tt, info)

	case *types.Basic:
		info.Kind = TypeKindBasic

	case *types.Pointer:
		info.Kind = TypeKindPointer
		info.ElemType = a.analyzeType(tt.Elem())

	case *types.Slice:
		info.Kind = TypeKindSlice
		info.ElemType = a.analyzeType(tt.Elem())

	case *types.Struct:
		info.Kind = TypeKindStruct
		a.analyzeStructFields(tt, info)

	default:
		// Maps, interfaces, channels, etc. are marked as unknown (unsupported)
		info.Kind = TypeKindUnknown
	}

	return info
}

// analyzeNamedType analyzes a named type.
func (a *Analyzer) analyzeNamedType(named *types.Named, info *TypeInfo) {
	obj := named.Obj()
	info.ID = TypeID{
		PkgPath: obj.Pkg().Path(),
		Name:    obj.Name(),
	}

	underlying := named.Underlying()

	switch ut := underlying.(type) {
	case *types.Struct:
		info.Kind = TypeKindStruct
		a.analyzeStructFields(ut, info)

	case *types.Basic:
		// Type alias for a basic type (e.g., type OrderStatus string)
		info.Kind = TypeKindAlias
		info.Underlying = a.analyzeType(ut)

	default:
		// External/opaque type (e.g., time.Time, or complex named types)
		// We check if it's from an external package
		if a.isExternalPackage(obj.Pkg().Path()) {
			info.Kind = TypeKindExternal
		} else {
			// Named type wrapping something else in our packages
			info.Kind = TypeKindAlias
			info.Underlying = a.analyzeType(ut)
		}
	}
}

// isExternalPackage returns true if the package is not in our analyzed set.
func (a *Analyzer) isExternalPackage(pkgPath string) bool {
	_, ok := a.graph.Packages[pkgPath]
	return !ok
}

// analyzeStructFields extracts fields from a struct type.
func (a *Analyzer) analyzeStructFields(st *types.Struct, info *TypeInfo) {
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)

		// Only process exported fields (per PLAN.md 80/20 rule)
		if !field.Exported() {
			continue
		}

		fieldInfo := FieldInfo{
			Name:     field.Name(),
			Exported: field.Exported(),
			Type:     a.analyzeType(field.Type()),
			Tag:      reflect.StructTag(st.Tag(i)),
			Embedded: field.Embedded(),
			Index:    i,
		}

		info.Fields = append(info.Fields, fieldInfo)
	}
}

// GetStruct returns the TypeInfo for a named struct by its fully qualified name.
// The typeName should be in the format "package.TypeName" (e.g., "store.Order").
func (a *Analyzer) GetStruct(pkgPath, typeName string) (*TypeInfo, error) {
	id := TypeID{PkgPath: pkgPath, Name: typeName}
	info := a.graph.GetType(id)
	if info == nil {
		return nil, fmt.Errorf("type %s not found", id)
	}
	if info.Kind != TypeKindStruct {
		return nil, fmt.Errorf("type %s is not a struct (kind: %s)", id, info.Kind)
	}
	return info, nil
}
