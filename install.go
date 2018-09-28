package main

import (
	"context"
	"errors"
)

// Install installs vendor dependencies from manifest.
func Install(ctx context.Context, verbose bool) error {
	if vendorExists() {
		return errors.New("vendor directory already exists")
	}

	for pkg, info := range manifest.Packages {
		_, isLocal := manifest.LocalPackages[pkg]

		if _, _, _, err := doImport(ctx, pkg, info.CommitHash, isLocal, false, false, verbose, true); err != nil {
			return err
		}
	}

	return nil
}
