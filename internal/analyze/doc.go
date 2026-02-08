// Package analyze provides package loading and type graph extraction.
//
// It uses golang.org/x/tools/go/packages with AST and go/types
// to build a canonical in-memory model of structs and their fields.
//
// Key types:
//   - TypeID: package import path + type name
//   - TypeInfo: describes kind (struct/basic/alias/pointer/slice/external)
//   - FieldInfo: describes field name, type, tags, and embedding
package analyze
