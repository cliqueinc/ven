package parse

import (
	"fmt"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

// GlideParser parses glide lock file.
type GlideParser struct{}

// glideLockFile represents glide lock file.
type glideLockFile struct {
	Imports []struct {
		Name    string
		Version string
	}
}

// Check checks whether repo uses glide.
func (p GlideParser) Check(repoPath string) bool {
	if _, err := os.Stat(repoPath + "/glide.lock"); err == nil {
		return true
	}

	return false
}

// Parse parses glide lock file for imports.
func (p GlideParser) Parse(repoPath string) ([]Package, error) {
	lockFile := glideLockFile{}
	data, err := ioutil.ReadFile(repoPath + "/glide.lock")
	if err != nil {
		return nil, fmt.Errorf("fail read glide.lock: %v", err)
	}

	if err := yaml.Unmarshal(data, &lockFile); err != nil {
		return nil, fmt.Errorf("fail unmarshal glide.lock file: %v", err)
	}

	packages := make([]Package, 0, len(lockFile.Imports))
	for _, info := range lockFile.Imports {
		packages = append(packages, Package{
			Name:       info.Name,
			CommitHash: info.Version,
		})
	}

	return packages, nil
}

func init() {
	Parsers["glide"] = GlideParser{}
}
