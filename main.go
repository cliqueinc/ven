package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		cancel()
	}()

	var (
		excludeBuilds, excludeDirs []string
		update, updateDeps         bool
		verbose                    bool
		constraint                 bool
	)

	var cmdGet = &cobra.Command{
		Use:   "get [packages to import]",
		Short: "Gets list of specified packages with its dependencies.",
		Long:  `get supports importing specific version of package (by tag, branch name or commit hash) and local packages`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				backupVendor bool
				isUpdate     = update || updateDeps
			)
			if isUpdate && vendorExists() {
				if err := copyDir(context.Background(), manifest.VendorPath, manifest.VendorPath+".orig"); err != nil {
					return fmt.Errorf("cannot backup vendor: %v", err)
				}
				backupVendor = true
			}

			newPkgs, err := Get(ctx, args, update, updateDeps, constraint, verbose)
			if err == nil {
				if backupVendor {
					if err := os.RemoveAll(manifest.VendorPath + ".orig"); err != nil {
						fmt.Printf("cannot delete backup: %v\n", err)
					}
				}
				return nil
			}

			// restore origin vendor after fail.
			if backupVendor {
				if err := os.RemoveAll(manifest.VendorPath); err != nil {
					return fmt.Errorf("cannot delete vendor: %v", err)
				}
				if err := os.Rename(manifest.VendorPath+".orig", manifest.VendorPath); err != nil {
					return fmt.Errorf("cannot rename vendor.orig: %v", err)
				}

				return err
			} else if !isUpdate {
				for _, pkg := range newPkgs {
					if err := os.RemoveAll(manifest.VendorPath + "/" + pkg); err != nil {
						fmt.Printf("cannot remove pkg (%s): %v\n", pkg, err)
					}
				}
			}

			return err
		},
	}
	cmdGet.Flags().BoolVarP(&verbose, "verbose", "v", false, "")
	cmdGet.Flags().BoolVarP(&constraint, "constraint", "c", false, "add package with version to constraint")
	cmdGet.Flags().BoolVarP(&update, "update", "u", false, "update package if exists")
	cmdGet.Flags().BoolVarP(&update, "update-deps", "", false, "update package dependencies")

	var cmdInstall = &cobra.Command{
		Use:   "install",
		Short: "Install installs vendor dependencies from manifest.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := Install(ctx, verbose)
			if !ctxCancelled(ctx) {
				return err
			}

			os.RemoveAll(manifest.VendorPath)
			return nil
		},
	}
	cmdInstall.Flags().BoolVarP(&verbose, "verbose", "v", false, "")

	var cmdInit = &cobra.Command{
		Use:   "init",
		Short: "Init defines a manifest for current project.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := Init(ctx, excludeBuilds, excludeDirs)
			if !ctxCancelled(ctx) {
				return err
			}

			return nil
		},
	}
	cmdInit.Flags().StringSliceVarP(&excludeBuilds, "exclude-builds", "", []string{"appenginevm", "appengine", "android", "integration", "ignore"}, "builds to exclude from import")
	cmdInit.Flags().StringSliceVarP(&excludeDirs, "exclude-dirs", "", []string{"test", "_fixture", "integration"}, "directories to exclude from import")

	var cmdFetch = &cobra.Command{
		Use:   "fetch",
		Short: "Fetch fetches dependencies for current project.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cannot get current directory: %v", err)
			}
			if vendorExists() {
				return errors.New("vendor directory already exists")
			}

			err = Fetch(ctx, strings.TrimPrefix(dir, os.Getenv("GOPATH")+"/src/"), update, verbose)
			if err != nil || ctxCancelled(ctx) {
				os.RemoveAll(manifest.VendorPath)
			}

			return err
		},
	}
	cmdFetch.Flags().BoolVarP(&verbose, "verbose", "v", false, "")

	var rootCmd = &cobra.Command{Use: "ven"}
	rootCmd.AddCommand(cmdInit, cmdFetch, cmdGet, cmdInstall)

	rootCmd.Execute()
}

func ctxCancelled(сtx context.Context) bool {
	select {
	case <-сtx.Done():
		return сtx.Err() == context.Canceled
	default:
		return false
	}
}
