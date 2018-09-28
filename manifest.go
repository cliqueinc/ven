package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

// Manifest describes manifest config file in a form simplier to work with.
type Manifest struct {
	VendorPath      string // defaults to "./vendor"
	ExcludeDir      map[string]struct{}
	ExcludeBuild    map[string]struct{}
	ExcludePackages map[string]struct{}
	LocalPackages   map[string]struct{}

	Constraints map[string]string
	Packages    map[string]Package
}

// ManifestYaml represents manifest config file.
type ManifestYaml struct {
	VendorPath      string   `yaml:"vendor_path"` // defaults to "./vendor"
	ExcludeDir      []string `yaml:"exclude_dir"`
	ExcludeBuild    []string `yaml:"exclude_build"`
	ExcludePackages []string `yaml:"exclude_packages"`
	LocalPackages   []string `yaml:"local_packages"`

	Constraints map[string]string
	Packages    map[string]PackageYaml
}

// Package describes manifest package.
type Package struct {
	Name        string
	Version     string
	CommitHash  string
	Subpackages map[string]struct{}
	Deps        map[string]struct{}
}

// PackageYaml describes manifest package in a yaml file.
type PackageYaml struct {
	Version     string
	CommitHash  string
	Subpackages []string
	Deps        []string
}

var manifest *Manifest

func init() {
	if _, err := os.Stat("Manifest.yml"); err != nil {
		if os.IsNotExist(err) {
			manifest = initManifest()
			return
		}
	}

	m, err := parseManifest()
	if err != nil {
		log.Fatalf("cannot read manifest: %v", err)
	}
	manifest = m
}

func (p Package) String() string {
	pkgInfo := fmt.Sprintf("commit %s", p.CommitHash)
	if p.Version != "" {
		pkgInfo += ", version: " + p.Version
	}

	return pkgInfo
}

// IsLocalPkg checks whether pkg is local.
func (m *Manifest) IsLocalPkg(pkg string) (string, bool) {
	parts := strings.Split(pkg, "/")
	for len(parts) != 0 {
		pkgPart := strings.Join(parts, "/")

		if _, ok := m.LocalPackages[pkgPart]; ok {
			return pkgPart, true
		}

		parts = parts[:len(parts)-1]
	}

	return "", false
}

// IsExcludedPkg checks whether pkg is excluded.
func (m *Manifest) IsExcludedPkg(pkg string) (string, bool) {
	parts := strings.Split(pkg, "/")
	for len(parts) != 0 {
		pkgPart := strings.Join(parts, "/")

		if _, ok := m.ExcludePackages[pkgPart]; ok {
			return pkgPart, true
		}

		parts = parts[:len(parts)-1]
	}

	return "", false
}

// GetPkgDeps gets pkg dependencies.
func (m *Manifest) GetPkgDeps(pkg string, pkgs map[string][]string) (string, []string, bool) {
	parts := strings.Split(pkg, "/")
	for len(parts) != 0 {
		pkgPart := strings.Join(parts, "/")

		if val, ok := pkgs[pkgPart]; ok {
			return pkgPart, val, true
		}

		parts = parts[:len(parts)-1]
	}

	return "", nil, false
}

// GetPkgConstraint gets pkg constraint if exists.
func (m *Manifest) GetPkgConstraint(pkg string) (string, string, bool) {
	parts := strings.Split(pkg, "/")
	for len(parts) != 0 {
		pkgPart := strings.Join(parts, "/")

		if val, ok := m.Constraints[pkgPart]; ok {
			return pkgPart, val, true
		}

		parts = parts[:len(parts)-1]
	}

	return "", "", false
}

// PkgExists checks whether pkg is already in manifest.
func (m *Manifest) PkgExists(pkg string) (existing Package, root string, exists bool) {
	parts := strings.Split(pkg, "/")
	for len(parts) != 0 {
		pkgPart := strings.Join(parts, "/")

		if val, ok := m.Packages[pkgPart]; ok {
			return val, pkgPart, true
		}

		parts = parts[:len(parts)-1]
	}

	return
}

func initManifest() *Manifest {
	return &Manifest{
		VendorPath:      "./vendor",
		ExcludeBuild:    make(map[string]struct{}),
		ExcludeDir:      make(map[string]struct{}),
		LocalPackages:   make(map[string]struct{}),
		ExcludePackages: make(map[string]struct{}),
		Constraints:     make(map[string]string),
		Packages:        make(map[string]Package),
	}
}

func parseManifest() (*Manifest, error) {
	cfg, m := &ManifestYaml{}, initManifest()
	data, err := ioutil.ReadFile("Manifest.yml")
	if err != nil {
		return nil, fmt.Errorf("fail read manifest: %v", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("fail unmarshal manifest config: %v", err)
	}
	if cfg.VendorPath == "" {
		cfg.VendorPath = "./vendor"
	}
	cfg.VendorPath = strings.TrimSuffix(cfg.VendorPath, "/")
	m.VendorPath = cfg.VendorPath
	for _, build := range cfg.ExcludeBuild {
		m.ExcludeBuild[build] = struct{}{}
	}
	for _, dir := range cfg.ExcludeDir {
		m.ExcludeDir[dir] = struct{}{}
	}
	for _, local := range cfg.LocalPackages {
		m.LocalPackages[local] = struct{}{}
	}
	for _, pkg := range cfg.ExcludePackages {
		m.ExcludePackages[pkg] = struct{}{}
	}
	for name, pkgYaml := range cfg.Packages {
		depsMap := make(map[string]struct{})
		for _, dep := range pkgYaml.Deps {
			depsMap[dep] = struct{}{}
		}
		subpkgsMap := make(map[string]struct{})
		for _, subpkg := range pkgYaml.Subpackages {
			subpkgsMap[subpkg] = struct{}{}
		}

		m.Packages[name] = Package{
			Name:        name,
			CommitHash:  pkgYaml.CommitHash,
			Version:     pkgYaml.Version,
			Subpackages: subpkgsMap,
			Deps:        depsMap,
		}
	}
	m.Constraints = cfg.Constraints

	return m, nil
}

func saveManifest() error {
	cfg := ManifestYaml{
		VendorPath:      manifest.VendorPath,
		ExcludeBuild:    make([]string, 0, 4),
		ExcludeDir:      make([]string, 0, 4),
		LocalPackages:   make([]string, 0, 4),
		ExcludePackages: make([]string, 0, 4),
		Constraints:     manifest.Constraints,
		Packages:        make(map[string]PackageYaml),
	}
	for build := range manifest.ExcludeBuild {
		cfg.ExcludeBuild = append(cfg.ExcludeBuild, build)
	}
	for dir := range manifest.ExcludeDir {
		cfg.ExcludeDir = append(cfg.ExcludeDir, dir)
	}
	for local := range manifest.LocalPackages {
		cfg.LocalPackages = append(cfg.LocalPackages, local)
	}
	for pkg := range manifest.ExcludePackages {
		cfg.ExcludePackages = append(cfg.ExcludePackages, pkg)
	}
	for name, pkg := range manifest.Packages {
		deps := make([]string, 0, len(pkg.Deps))
		for dep := range pkg.Deps {
			deps = append(deps, dep)
		}
		sort.Strings(deps)
		subpkgs := make([]string, 0, len(pkg.Subpackages))
		for subpkg := range pkg.Subpackages {
			subpkgs = append(subpkgs, subpkg)
		}
		sort.Strings(subpkgs)

		cfg.Packages[name] = PackageYaml{
			CommitHash:  pkg.CommitHash,
			Version:     pkg.Version,
			Subpackages: subpkgs,
			Deps:        deps,
		}
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("fail marshal manifest config: %v", err)
	}

	if err := os.Rename("Manifest.yml", "Manifest.orig.yml"); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot backup manifest file: %v", err)
		}
	} else {
		defer os.Rename("Manifest.orig.yml", "Manifest.yml")
	}

	f, err := os.Create("Manifest.yml")
	if err != nil {
		return fmt.Errorf("cannot open manifest file: %v", err)
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return fmt.Errorf("cannot write to manifest file: %v", err)
	}
	os.Remove("Manifest.orig.yml")

	return nil
}
