package common

import "path"

// PkgAlias returns the package alias (last element of path) for a given package path.
// Returns empty string if pkgPath is empty.
func PkgAlias(pkgPath string) string {
	if pkgPath == "" {
		return ""
	}

	return path.Base(pkgPath)
}
