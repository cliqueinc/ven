package parse

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// GovendorParser parses govendor file.
type GovendorParser struct{}

// govendorFile represents govendor file.
type govendorFile struct {
	Package []struct {
		Path     string `json:"path"`
		Revision string `json:"revision"`
	} `json:"package"`
}

// Check checks whether repo uses glide.
func (p GovendorParser) Check(repoPath string) bool {
	if _, err := os.Stat(repoPath + "/vendor/vendor.json"); err == nil {
		return true
	}

	return false
}

// Parse parses govendor file for imports.
func (p GovendorParser) Parse(repoPath string) ([]Package, error) {
	cfgFile := govendorFile{}
	data, err := ioutil.ReadFile(repoPath + "/vendor/vendor.json")
	if err != nil {
		return nil, fmt.Errorf("fail read vendor.json file: %v", err)
	}

	if err := json.Unmarshal(data, &cfgFile); err != nil {
		return nil, fmt.Errorf("fail unmarshal vendor.json file: %v", err)
	}

	packages := make([]Package, 0, len(cfgFile.Package))
	for _, info := range cfgFile.Package {
		packages = append(packages, Package{
			Name:       info.Path,
			CommitHash: info.Revision,
		})
	}

	return packages, nil
}

func init() {
	Parsers["govendor"] = GovendorParser{}
}
