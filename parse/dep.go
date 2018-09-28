package parse

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pelletier/go-toml"
)

// DepParser parses go dep file.
type DepParser struct{}

// depFile represents dep file.
type depFile struct {
	Projects []struct {
		Name     string `toml:"name"`
		Revision string `toml:"revision"`
	} `toml:"projects"`
}

// Check checks whether repo uses go dep.
func (p DepParser) Check(repoPath string) bool {
	if _, err := os.Stat(repoPath + "/Gopkg.lock"); err == nil {
		return true
	}

	return false
}

// Parse parses dep file for imports.
func (p DepParser) Parse(repoPath string) ([]Package, error) {
	cfgFile := depFile{}
	data, err := ioutil.ReadFile(repoPath + "/Gopkg.lock")
	if err != nil {
		return nil, fmt.Errorf("fail read Gopkg.lock file: %v", err)
	}

	if err := toml.Unmarshal(data, &cfgFile); err != nil {
		return nil, fmt.Errorf("fail unmarshal Gopkg.lock file: %v", err)
	}

	packages := make([]Package, 0, len(cfgFile.Projects))
	for _, info := range cfgFile.Projects {
		packages = append(packages, Package{
			Name:       info.Name,
			CommitHash: info.Revision,
		})
	}

	return packages, nil
}

func init() {
	Parsers["dep"] = DepParser{}
}
