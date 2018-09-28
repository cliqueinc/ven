package main

import (
	"context"
	"errors"
	"os"
)

// Init inits manifest for a current project.
func Init(ctx context.Context, excludeBuilds, excludeDirs []string) error {
	if _, err := os.Stat("Manifest.yml"); err == nil || !os.IsNotExist(err) {
		return errors.New("manifest already exists")
	}

	for _, build := range excludeBuilds {
		manifest.ExcludeBuild[build] = struct{}{}
	}
	for _, dir := range excludeDirs {
		manifest.ExcludeDir[dir] = struct{}{}
	}

	if err := saveManifest(); err != nil {
		return err
	}

	return nil
}
