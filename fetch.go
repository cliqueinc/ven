package main

import (
	"context"
	"fmt"
)

// Fetch fetches dependencies for current project.
func Fetch(ctx context.Context, pkg string, update, verbose bool) error {
	if _, err := getPkgImportsFromPopularVendorTools(pkg, "./", verbose); err != nil && verbose {
		fmt.Println(err)
	}

	_, _, depsMap, err := getPkgImports(pkg, Package{}, nil, "./", true, true, true, verbose)
	if err != nil {
		return fmt.Errorf("failed get imports for a project: %v", err)
	}

	for importRoot, importDeps := range depsMap {
		opts := ImportOptions{
			Subpackages: importDeps,
			Update:      update,
			UpdateDeps:  update,
		}

		if err := importPackage(ctx, importRoot, opts, verbose); err != nil {
			return err
		}
	}
	if err := saveManifest(); err != nil {
		return err
	}

	return nil
}
