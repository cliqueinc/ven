package main

import (
	"context"
	"strings"
)

// Get gets list of specified packages with its dependencies.
func Get(ctx context.Context, pkgs []string, update, updateDeps, constraint, verbose bool) ([]string, error) {
	var newPkgs []string

	for _, pkg := range pkgs {
		var isLocal bool
		if strings.HasPrefix(pkg, "file://") {
			isLocal = true
			pkg = strings.TrimPrefix(pkg, "file://")
		}

		var version string
		if i := strings.Index(pkg, "@"); i != -1 && i != len(pkg)-1 {
			pkg, version = pkg[:i], pkg[i+1:]
			if constraint {
				manifest.Constraints[pkg] = version
			}
		}
		opts := ImportOptions{
			FetchAll:   pkg == getPkgRoot(pkg),
			Local:      isLocal,
			Update:     update,
			UpdateDeps: updateDeps,
			Version:    version,
		}

		if err := importPackage(ctx, pkg, opts, verbose); err != nil {
			return newPkgs, err
		}
	}
	if err := saveManifest(); err != nil {
		return newPkgs, err
	}

	return newPkgs, nil
}
