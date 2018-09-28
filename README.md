# Ven package manager

## What is Ven and why have we built it?

As of 2017, golang did not ship with a proper, production-ready package
manager. There are many other alternatives, but all that we tried had
serious issues (details coming later).

We decided to build our own to accommodate a very simple workflow:
- Store dependencies locally and check them in
- Remove non-go dependency files in most cases to avoid storing a ton
  of files

## Installing Ven

Clone the repo into the proper location in your go path, like:

`go_workspace/src/github.com/cliqueinc/ven`

Install the `ven` command
`$ cd ven && go install`

**The `ven` command will not be available in your terminal unless you
have go_workspace/bin in your PATH.** Please see
[How to write go code](https://golang.org/doc/code.html) for more info.

## Usage

Once you have ven installed and it is available in your terminal, simply
go to the root directory of your own go project and add some dependencies
using `ven get github.com/someuser/someproject`! Enjoy.

```
Usage:
  ven [command]

Available Commands:
  fetch       Fetch fetches dependencies for current project.
  get         Gets list of specified packages with its dependencies.
  help        Help about any command
  init        Init defines a manifest for current project.
```

## Commands

- Get

  Get supports importing specific version of package (by tag, branch name or commit hash) and local packages

  ```
  Usage:
    ven get [packages to import] [flags]

  Flags:
    -h, --help          help for get
    -u, --update        update package if exists
        --update-deps   update package dependencies
  ```

- Init

  Init defines a manifest for current project.
  ```
  Usage:
  ven init [flags]

  Flags:
        --exclude-builds stringSlice   builds to exclude from import (default [appenginevm,appengine,android,integration,ignore])
        --exclude-dirs stringSlice     directories to exclude from import (default [cmd])
    -h, --help                         help for init
  ```

  - Fetch

  Fetch fetches dependencies for current project.
  ```
  Usage:
  ven fetch
  ```

## Manifest sample:

```
exclude_dir:
- cmd
exclude_build:
- appenginevm
- appengine
- android
- integration
- ignore
local_packages: []
constraints:
  github.com/labstack/echo: v2.2.0
packages:
  github.com/davecgh/go-spew:
    version: v1.1.0
    commithash: adab96458c51a58dc1783b3335dcce5461522e75
    deps: []
  github.com/dgrijalva/jwt-go:
    version: v3.0.0
    commithash: a539ee1a749a2b895533f979515ac7e6e0f5b650
    deps: []
  github.com/klauspost/compress:
    version: v1.2.1
    commithash: f3dce52e0576655d55fd69e74b63da96ad1108f3
    deps:
    - github.com/klauspost/cpuid
```

- `exclude_dir` - array of directories to exclude from import.
- `exclude_build` - array of build tags to exclude from searching for dependencies, for example `windows`, `appengine`.
- `local_packages` - list of packages to search in a local filesystem.
- `constraints` - constraints for a specific packages, if not set, the latest version of a package will be loaded, or the one specified in a get command.
- `packages` - list of downloaded packages.


## Upgrading Ven
Simply `git pull` in the ven repo you checked out previous and run `go install`
again.
