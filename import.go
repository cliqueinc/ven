package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cliqueinc/ven/parse"
	"golang.org/x/tools/go/vcs"
)

var (
	newPkgs    = make([]string, 0, 10)
	cachedPkgs = make(map[string]struct{})
	// cachedConstraints keeps desired, but not required pkg versions for load (taken from other package managers)
	cachedConstraints = make(map[string]string)
	cachedExcluded    = make(map[string]struct{})
)

// ImportOptions describes import options.
type ImportOptions struct {
	// pkg version
	Version string

	// specified whether a pkg is local
	Local bool

	// Subpackages describes pkg nested subpackages that needed to be fetched
	Subpackages []string

	// if set to true, all pkg suppackages will be scanned for dependencies, othervise only used packages will be scanned.
	FetchAll bool

	// if set to true, ven will try to update existing package, in case if newest version differs from existing one.
	Update bool

	// Tells whether pkg dependencies need to be updated, othervise only new pkgs will be downloaded.
	UpdateDeps bool
}

// importPackage imports package with it's dependencies.
func importPackage(ctx context.Context, pkg string, opts ImportOptions, verbose bool) error {
	var (
		updatePkgImports bool
		performImport    = true
		isNewPkg         = true
		isUpdate         = opts.Update || opts.UpdateDeps
		isLocal          = opts.Local
		version          = opts.Version
		newSubpkgs       = opts.Subpackages
		versionRequired  bool
	)

	rootPkg := getPkgRoot(pkg)
	if root, exclude := manifest.IsExcludedPkg(rootPkg); exclude {
		if _, ok := cachedExcluded[root]; ok {
			return nil
		}
		if verbose {
			fmt.Printf("pkg (%s) is excluded from import\n", root)
		}
		cachedExcluded[root] = struct{}{}
		return nil
	}
	if _, local := manifest.IsLocalPkg(rootPkg); local {
		isLocal = true
	}
	if _, constraintVersion, exists := manifest.GetPkgConstraint(rootPkg); exists && constraintVersion != "" {
		versionRequired = true
		if version == "" {
			version = constraintVersion
		} else if version != constraintVersion {
			return fmt.Errorf("pkg (%s): pkg has a constraint (%s), can't import version (%s)", rootPkg, constraintVersion, version)
		}
	} else if v := cachedConstraints[rootPkg]; version == "" && v != "" {
		version = v
	}

	var info Package
	if existing, root, exists := manifest.PkgExists(rootPkg); exists {
		rootPkg = root
		if pkg != rootPkg {
			newSubpkgs = append(newSubpkgs, pkg)
		}
		if len(newSubpkgs) != 0 {
			var i int
			for _, subpkg := range newSubpkgs {
				if _, ok := existing.Subpackages[subpkg]; !ok {
					if !dirIsExcluded(subpkg) {
						newSubpkgs[i] = subpkg
						i++
					}
				}
			}
			newSubpkgs = newSubpkgs[:i]
			updatePkgImports = len(newSubpkgs) != 0
		}

		if _, ok := cachedPkgs[root]; ok {
			if !updatePkgImports {
				return nil
			}
		}

		isNewPkg = false
		if !opts.Update {
			if !updatePkgImports {
				if verbose {
					fmt.Printf("pkg (%s): already in manifest with version: (%s)\n", root, existing)
				}
				return nil
			}
			performImport = false
		}
		if version != "" && version == existing.Version {
			if !updatePkgImports {
				if verbose {
					fmt.Printf("pkg (%s): already up to date\n", root)
				}
				return nil
			}
			performImport = false
		}

		if opts.Update {
			if err := os.RemoveAll(fmt.Sprintf("%s/%s", manifest.VendorPath, root)); err != nil {
				return fmt.Errorf("pkg (%s): fail remove existing pkg", root)
			}
		}
		info = existing
	}

	if performImport {
		root, pkgInfo, _, err := doImport(ctx, rootPkg, version, isLocal, opts.Update, true, verbose, versionRequired)
		if root != "" && ctxCancelled(ctx) && !isUpdate {
			newPkgs = append(newPkgs, root)
		}
		if err != nil {
			return err
		}

		if !isNewPkg {
			subpkgs, err := getPkgSubpackages(root, fmt.Sprintf("%s/%s", manifest.VendorPath, rootPkg), verbose)
			if err != nil {
				return err
			}

			subpkgsMap := make(map[string]struct{})
			for _, subpkg := range subpkgs {
				subpkgsMap[subpkg] = struct{}{}
			}

			deprecatedPkgs := make([]string, 0, 4)
			for subpkg := range info.Subpackages {
				if _, ok := subpkgsMap[subpkg]; !ok {
					deprecatedPkgs = append(deprecatedPkgs, subpkg)
				}
			}

			// some subpackages were removed during update, need to check manifest on usage them by other pkgs
			if len(deprecatedPkgs) != 0 {
				var msgs []string
				for pName, p := range manifest.Packages {
					for _, deprecated := range deprecatedPkgs {
						if _, ok := p.Deps[deprecated]; ok {
							msgs = append(msgs, fmt.Sprintf("pkg (%s) is using subpackage (%s), which is deprecated in version: (%s)", pName, deprecated, pkgInfo))
						}
					}
				}
				if len(msgs) != 0 {
					return fmt.Errorf("%s. You should consider updating these packages first to solve dependency conflicts", strings.Join(msgs, "; "))
				}
			}

			for _, deprecated := range deprecatedPkgs {
				delete(info.Subpackages, deprecated)
			}
			pkgInfo.Subpackages = info.Subpackages
		}
		rootPkg = root
		info = pkgInfo

		if verbose {
			fmt.Println(info)
		}
	}
	if !isUpdate && isNewPkg {
		newPkgs = append(newPkgs, rootPkg)
	}

	if ctxCancelled(ctx) {
		return ctx.Err()
	}

	imports, localSubpkgs, depsMap, err := getPkgImports(rootPkg, info, newSubpkgs, fmt.Sprintf("%s/%s", manifest.VendorPath, rootPkg), isNewPkg, opts.FetchAll, false, verbose)
	if err != nil {
		return fmt.Errorf("pkg (%s): failed get imports: %v", pkg, err)
	}

	for _, i := range imports {
		info.Deps[i] = struct{}{}
	}
	for _, subpkg := range localSubpkgs {
		if subpkg != rootPkg {
			info.Subpackages[subpkg] = struct{}{}
		}
	}

	manifest.Packages[rootPkg] = info
	cachedPkgs[rootPkg] = struct{}{}
	if isLocal {
		if _, ok := manifest.LocalPackages[rootPkg]; !ok {
			manifest.LocalPackages[rootPkg] = struct{}{}
		}
	}

	for importRoot, importSubpkgs := range depsMap {
		importOpts := ImportOptions{
			Update:      opts.UpdateDeps,
			UpdateDeps:  opts.UpdateDeps,
			Subpackages: importSubpkgs,
		}

		if err := importPackage(ctx, importRoot, importOpts, verbose); err != nil {
			return err
		}
	}

	return nil
}

// doImport imports package only. Returns pkg root, pkg dependencies and an error if occur.
func doImport(ctx context.Context, pkg, version string, isLocal, update, fetchDeps, verbose, versionRequired bool) (root string, info Package, deps []string, pkgErr error) {
	var commit, commitVersion string
	if isLocal {
		if verbose {
			fmt.Printf("pkg %s is local, clonning...\n", pkg)
		}
		dirPath := fmt.Sprintf("%s/src/%s", os.Getenv("GOPATH"), pkg)
		if _, err := os.Stat(dirPath); err != nil {
			if os.IsNotExist(err) {
				pkgErr = fmt.Errorf("cannot detect vcs version of package %s: %v", pkg, err)
				return
			}
			pkgErr = fmt.Errorf("cannot get info about local pkg %s: %v", dirPath, err)
			return
		}

		rootPath := fmt.Sprintf("%s/src/", os.Getenv("GOPATH"))
		vcs, rootPkg, err := vcs.FromDir(dirPath, rootPath)
		if err != nil {
			pkgErr = fmt.Errorf("cannot detect vcs version of package %s: %v", pkg, err)
			return
		}
		root = rootPkg
		if vcs.Cmd != "git" {
			pkgErr = fmt.Errorf("pkg (%s): ven supports only git repos", pkg)
			return
		}
		pkg = root

		pkgPath := fmt.Sprintf("%s/src/%s", os.Getenv("GOPATH"), pkg)
		commit, commitVersion, err = cloneLocalPkg(ctx, pkg, version, pkgPath, fmt.Sprintf("%s/%s", manifest.VendorPath, pkg))
		if err != nil {
			pkgErr = fmt.Errorf("cannot clone local package %s: %v", pkg, err)
			return
		}
	} else {
		repoRoot, err := vcs.RepoRootForImportPath(pkg, false)
		if err != nil {
			pkgErr = fmt.Errorf("pkg (%s): cannot detect pkg repository: %v", pkg, err)
			return
		}
		if repoRoot.VCS.Cmd != "git" {
			pkgErr = fmt.Errorf("pkg (%s): ven supports only git repos", pkg)
			return
		}

		pkg = repoRoot.Root
		root = pkg
		if verbose {
			fmt.Println(pkg, version)
		}
		commit, commitVersion, err = cloneRepo(ctx, pkg, version, repoRoot.Repo+".git", fmt.Sprintf("%s/%s", manifest.VendorPath, pkg), versionRequired)
		if err != nil {
			pkgErr = fmt.Errorf("pkg (%s): cannot clone repo: %v", pkg, err)
			return
		}
	}
	vendorPath := fmt.Sprintf("%s/%s", manifest.VendorPath, pkg)

	if fetchDeps {
		pkgs, err := getPkgImportsFromPopularVendorTools(pkg, vendorPath, verbose)
		if err != nil {
			if verbose {
				fmt.Println(err)
			}
		}
		deps = pkgs
	}
	if err := filterNonGoFiles(vendorPath); err != nil {
		pkgErr = fmt.Errorf("failed filter pkg (%s): %v", pkg, err)
		return
	}

	pkgVersion := version
	if version == "" {
		pkgVersion = commitVersion
	}

	info = Package{
		CommitHash:  commit,
		Version:     pkgVersion,
		Deps:        make(map[string]struct{}),
		Subpackages: make(map[string]struct{}),
	}

	return
}

func cloneLocalPkg(ctx context.Context, pkg, version, pkgPath, destPath string) (string, string, error) {
	if err := copyDir(ctx, pkgPath, destPath); err != nil {
		return "", "", err
	}

	return checkoutRepo(ctx, version, destPath, true)
}

// cloneRepo clones repo with specific reference
func cloneRepo(ctx context.Context, pkg, version, repo, dir string, versionRequired bool) (string, string, error) {
	var outb, errb bytes.Buffer

	cmd := exec.CommandContext(ctx, "git", "clone", repo, dir)
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	if err := cmd.Run(); err != nil {
		os.Stdout.Write(outb.Bytes())
		os.Stderr.Write(errb.Bytes())
		return "", "", err
	}

	return checkoutRepo(ctx, version, dir, versionRequired)
}

func checkoutRepo(ctx context.Context, version, repoPath string, versionRequired bool) (string, string, error) {
	var outb, errb bytes.Buffer

	if version != "" {
		cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", version)
		cmd.Stdout = &outb
		cmd.Stderr = &errb

		if err := cmd.Run(); err != nil && versionRequired {
			os.Stdout.Write(outb.Bytes())
			os.Stderr.Write(errb.Bytes())
			return "", "", err
		}

		outb.Reset()
		errb.Reset()
	}

	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "HEAD")
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	if err := cmd.Run(); err != nil {
		os.Stdout.Write(outb.Bytes())
		os.Stderr.Write(errb.Bytes())
		return "", "", err
	}

	commit := strings.TrimSpace(outb.String())

	outb.Reset()
	errb.Reset()

	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "describe", "--tags", "--abbrev=0")
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	if err := cmd.Run(); err != nil {
		return commit, "", nil
	}

	return commit, strings.TrimSpace(outb.String()), nil
}

func filterNonGoFiles(dir string) error {
	var dirs []string

	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if err != nil || f == nil {
			if os.IsNotExist(err) {
				return nil
			}
			if _, err = os.Stat(path); os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if f.IsDir() {
			if dirIsExcluded(path) {
				if err := os.RemoveAll("./" + path); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("fail delete (%s): %v", path, err)
				}
				return nil
			}
			dirs = append(dirs, path)

			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			// go package may use cgo or assembler files.
			excludeExt := []string{"s", "S", "asm", "h", "o", "c", "cc"}
			for _, ext := range excludeExt {
				if strings.HasSuffix(path, "."+ext) {
					return nil
				}
			}
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("fail delete (%s): %v", path, err)
			}
		}
		return nil
	})

	for _, d := range dirs {
		files, err := ioutil.ReadDir(d)
		if err != nil {
			continue
		}
		if len(files) == 0 {
			os.Remove(d)
		}
	}

	return err
}

func dirIsExcluded(path string) bool {
	dirName := path
	if ind := strings.LastIndex(path, "/"); ind != -1 {
		dirName = path[ind+1:]
	}
	if dirName == "vendor" || strings.HasPrefix(dirName, ".") || strings.HasPrefix(dirName, "_") {
		return true
	}
	excludes := []string{"testdata", "_testdata"}
	for excl := range manifest.ExcludeDir {
		excludes = append(excludes, excl)
	}
	for _, excl := range excludes {
		if strings.HasSuffix(path, "/"+excl) || strings.Contains("/"+path, "/"+excl+"/") {
			return true
		}
	}
	return false
}

func getPkgImportsFromPopularVendorTools(pkg, dir string, verbose bool) ([]string, error) {
	// first try to parse deps from popular vendoring tools.
	for pName, p := range parse.Parsers {
		if !p.Check(dir) {
			continue
		}

		pkgs, err := p.Parse(dir)
		if err != nil {
			return nil, fmt.Errorf("pkg (%s): failed parse (%s) config file: %v", pkg, pName, err)
		}
		if verbose {
			fmt.Printf("pkg (%s): detected %s vendoring\n", pkg, pName)
		}

		imports := make([]string, 0, len(pkgs))
		for _, subPkg := range pkgs {
			if _, exist := cachedConstraints[subPkg.Name]; !exist {
				cachedConstraints[subPkg.Name] = subPkg.CommitHash
			}
			imports = append(imports, subPkg.Name+"@"+subPkg.CommitHash)
		}

		return imports, nil
	}

	return nil, nil
}

// go list -f '{{join .Deps "\n"}}' |  xargs go list -f '{{if not .Standard}}{{.ImportPath}}{{end}}'
func getPkgImports(pkg string, info Package, subpkgs []string, dir string, isNewPkg, fetchAll bool, parseMain, verbose bool) ([]string, []string, map[string][]string, error) {
	importsMap := make(map[string]struct{})

	var localPkgs []string
	if fetchAll {
		locals, err := walkImports(pkg, info, dir, fetchAll, importsMap, parseMain, verbose)
		if err != nil {
			return nil, nil, nil, err
		}
		localPkgs = locals
	} else {
		var includesMainDir bool
		dirsToWalk := make([]string, 0, len(subpkgs))
		for _, d := range subpkgs {
			if !includesMainDir && (d == dir || d == pkg) {
				includesMainDir = true
				dirsToWalk = append(dirsToWalk, d)
			} else if !dirIsExcluded(d) {
				dirsToWalk = append(dirsToWalk, d)
			}
		}
		if !includesMainDir && isNewPkg {
			if dirsToWalk == nil {
				dirsToWalk = []string{}
			}
			dirsToWalk = append(dirsToWalk, pkg)
		}
		walkMap := make(map[string]struct{})
		for len(dirsToWalk) != 0 {
			nextDirs := make([]string, 0, 4)
			for _, dir := range dirsToWalk {
				locals, err := walkImports(pkg, info, manifest.VendorPath+"/"+dir, fetchAll, importsMap, parseMain, verbose)
				if err != nil {
					return nil, nil, nil, err
				}
				walkMap[dir] = struct{}{}

				for _, l := range locals {
					if _, ok := walkMap[l]; !ok {
						nextDirs = append(nextDirs, l)
					}
				}
			}

			dirsToWalk = nextDirs
		}
		localPkgs = make([]string, 0, len(walkMap))
		for subpkg := range walkMap {
			localPkgs = append(localPkgs, subpkg)
		}
	}

	imports := make([]string, 0, len(importsMap))
	for i := range importsMap {
		imports = append(imports, i)
	}
	sort.Strings(imports)

	rootPkgsMap := make(map[string][]string)
	for _, i := range imports {
		rootPkg := getPkgRoot(i)
		if _, root, exists := manifest.PkgExists(rootPkg); exists {
			rootPkg = root
		}
		if root, subpkgs, exists := manifest.GetPkgDeps(rootPkg, rootPkgsMap); exists {
			rootPkgsMap[root] = append(subpkgs, i)
			continue
		}
		rootPkgsMap[rootPkg] = []string{i}
	}

	return imports, localPkgs, rootPkgsMap, nil
}

func getPkgSubpackages(pkg, dir string, verbose bool) ([]string, error) {
	subpkgs := make([]string, 0, 4)
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !f.IsDir() || path == "" {
			return nil
		}
		subpkgs = append(subpkgs, strings.TrimPrefix(path, manifest.VendorPath+"/"))

		return nil
	})
	if err != nil {
		return nil, err
	}

	return subpkgs, err
}

func walkImports(pkg string, info Package, dir string, scanAll bool, importsMap map[string]struct{}, parseMain, verbose bool) ([]string, error) {
	fset := token.NewFileSet()

	locals := make([]string, 0, 4)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory %s not found, perhaps version of pkg (%s %s) is not compatible with code", dir, pkg, info)
		}
		return nil, err
	}
FilesLoop:
	for _, f := range files {
		path := path.Join(dir, f.Name())
		if f.IsDir() {
			if dirIsExcluded(path) {
				continue
			}
			if scanAll {
				localPkgs, err := walkImports(pkg, info, path, scanAll, importsMap, parseMain, verbose)
				if err != nil {
					return nil, err
				}
				locals = append(locals, localPkgs...)
			}
			continue
		}
		if !strings.HasSuffix(f.Name(), ".go") {
			continue
		}

		parsedFile, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			if verbose {
				fmt.Printf("fail parse imports for file (%s): %v\n", path, err)
			}
			continue
		}

		if len(parsedFile.Imports) == 0 {
			continue
		}
		if !parseMain && parsedFile.Name != nil && parsedFile.Name.Name == "main" {
			continue
		}
		if len(manifest.ExcludeBuild) != 0 {
			for excl := range manifest.ExcludeBuild {
				if strings.HasSuffix(f.Name(), "_"+excl+".go") {
					if verbose {
						fmt.Printf("file (%s) with build tag (%s) excluded\n", f.Name(), excl)
					}
					continue FilesLoop
				}
			}

			file, err := os.Open(path)
			if err != nil {
				return nil, fmt.Errorf("fail open file %s: %v", path, err)
			}
			scanner := bufio.NewScanner(file)
			var line string

		ScanLoop:
			for scanner.Scan() {
				line = scanner.Text()
				if strings.HasPrefix(line, "package") {
					break
				}
				buildIndex := strings.Index(line, "+build")
				if !strings.HasPrefix(line, "//") || buildIndex == -1 || buildIndex+len("+build ") >= len(line) {
					continue
				}

				for _, build := range strings.Split(line[buildIndex+len("+build "):], " ") {
					if _, ok := manifest.ExcludeBuild[build]; !ok {
						continue ScanLoop
					}
				}
				if verbose {
					fmt.Printf("file (%s) with build tag(s) (%s) excluded\n", path, line[buildIndex+len("+build "):])
				}
				file.Close()
				continue FilesLoop
			}
			file.Close()
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("fail read file %s: %v", path, err)
			}
		}

		for _, i := range parsedFile.Imports {
			iVal := strings.Replace(i.Path.Value, "\"", "", -1)

			// from std
			if !strings.Contains(iVal, "/") {
				continue
			}
			if strings.HasPrefix(iVal, "./") || strings.HasPrefix(iVal, "../") {
				continue
			}
			// not external
			if !strings.Contains(iVal[:strings.Index(iVal, "/")], ".") {
				continue
			}
			// subpkg of current pkg
			if pkg != "" && strings.HasPrefix(iVal, pkg) {
				locals = append(locals, iVal)
				continue
			}
			pkgRoot := pkg
			parts := strings.Split(pkg, "/")
			// for the case if we call vendor not from repo root, but for instance from ./cmd,
			// that has cmd specific deps, we don't want to import project in vendor.
			if len(parts) > 3 {
				pkgRoot = strings.Join(parts[:3], "/")
			}

			if pkg != "" && (strings.HasPrefix(iVal, pkg) || strings.HasPrefix(iVal, pkgRoot)) {
				locals = append(locals, iVal)
				continue
			}

			if _, exists := importsMap[iVal]; !exists {
				importsMap[iVal] = struct{}{}
			}
		}
	}

	return locals, nil
}

// getPkgRoot detects pkg root from commonly used domains.
func getPkgRoot(pkg string) string {
	parts := strings.Split(pkg, "/")
	if len(parts) < 3 {
		return pkg
	}

	switch parts[0] {
	case "github.com", "gopkg.in", "golang.org", "gitlab.com":
		return strings.Join(parts[:3], "/")
	default:
		return pkg
	}
}
