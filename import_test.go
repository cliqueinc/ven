package main

import (
	"reflect"
	"testing"
)

func Test_getPkgImports(t *testing.T) {
	manifest.ExcludeBuild = map[string]struct{}{"go1.2": {}, "windows": {}, "386": {}, "integration": {}}
	manifest.ExcludeDir = map[string]struct{}{"excluded": {}}

	got, _, _, err := getPkgImports("github.com/pkg/path", Package{}, nil, "testdata", true, true, false, true)
	if err != nil {
		t.Fatalf("getPkgImports() error: %v", err)
	}

	expected := []string{"github.com/asaskevich/govalidator", "github.com/asaskevich/some", "github.com/asaskevich/wrong", "github.com/golang/dep/internal/gps", "github.com/pelletier/go-toml", "github.com/pkg/errors"}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("getPkgImports() = %v, want %v", got, expected)
	}
}
