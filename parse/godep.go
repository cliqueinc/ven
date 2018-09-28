package parse

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

// GodepParser parses glgodeps json file.
type GodepParser struct{}

// godepLockFile represents glide lock file.
type godepLockFile struct {
	Deps []struct {
		ImportPath string
		Rev        string
	}
}

// Check if necessary lock file for curent parser exists
func (p GodepParser) Check(repoPath string) bool {
	depsJSONPath := path.Join(repoPath, "Godeps/Godeps.json")
	if _, err := os.Stat(depsJSONPath); err != nil {
		return false
	}
	return true
}

// Parse packages
func (p GodepParser) Parse(repoPath string) ([]Package, error) {
	lockFile := godepLockFile{}
	data, err := ioutil.ReadFile(repoPath + "/Godeps/Godeps.json")
	if err != nil {
		return nil, fmt.Errorf("fail read Godeps.json: %v", err)
	}

	if err := json.Unmarshal(data, &lockFile); err != nil {
		return nil, fmt.Errorf("fail unmarshal Godeps.json file: %v", err)
	}

	packages := make([]Package, 0, len(lockFile.Deps))
	for _, info := range lockFile.Deps {
		packages = append(packages, Package{
			Name:       info.ImportPath,
			CommitHash: info.Rev,
		})
	}

	return packages, nil
}

func init() {
	Parsers["godep"] = GodepParser{}
}
