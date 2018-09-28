package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

// copyDir copies a directory recursively.
func copyDir(ctx context.Context, path, destPath string) error {
	if ctxCancelled(ctx) {
		return errors.New("cancelled")
	}

	pathInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !pathInfo.IsDir() {
		return errors.New("path is not a directory")
	}

	if _, err := os.Stat(destPath); err == nil || !os.IsNotExist(err) {
		return errors.New("directory already exists")
	}

	if err = os.MkdirAll(destPath, pathInfo.Mode()); err != nil {
		return err
	}

	filesInfo, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	if ctxCancelled(ctx) {
		return errors.New("cancelled")
	}

	for _, i := range filesInfo {
		fPath := path + "/" + i.Name()
		dPath := destPath + "/" + i.Name()
		if i.IsDir() {
			if err := copyDir(ctx, fPath, dPath); err != nil {
				return fmt.Errorf("cannot copy dir %s: %v", fPath, err)
			}

			continue
		}

		if err := copyFile(ctx, fPath, dPath); err != nil {
			return fmt.Errorf("cannot copy file %s: %v", fPath, err)
		}
	}

	return nil
}

// copyFile copies file to destPath.
func copyFile(ctx context.Context, path, destPath string) error {
	if ctxCancelled(ctx) {
		return errors.New("cancelled")
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if err := os.Chmod(destPath, info.Mode()); err != nil {
		return err
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	if _, err := destFile.Write(data); err != nil {
		return err
	}

	return nil
}

func vendorExists() bool {
	if _, err := os.Stat(manifest.VendorPath); err == nil || !os.IsNotExist(err) {
		return true
	}

	return false
}
