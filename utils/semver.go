package utils

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"

	coreos "github.com/coreos/go-semver/semver"
)

const (
	writeFilePermission = 0644
)

type Version struct {
	*coreos.Version
	filename string
}

func ReadVersion() (*Version, error) {
	bytes, err := os.ReadFile("version")
	if err == nil {
		return &Version{
			Version:  coreos.New(string(bytes)),
			filename: "version",
		}, nil
	}
	file, err := parser.ParseFile(token.NewFileSet(), "version.go", nil, 0)
	if err == nil {
		return &Version{
			Version: &coreos.Version{
				Major:      getValue(0, file),
				Minor:      getValue(1, file),
				Patch:      getValue(2, file),
				PreRelease: "SNAPSHOT",
				Metadata:   "",
			},
			filename: "version.go",
		}, nil
	}
	return nil, err
}

func (v *Version) Save() error {
	if v.filename == "version" {
		return os.WriteFile(v.filename, []byte(v.String()), writeFilePermission)
	}
	data := fmt.Sprintf("package main\n\nvar Major = %d\nvar Minor = %d\nvar Patch = %d", v.Major, v.Minor, v.Patch)
	return os.WriteFile(v.filename, []byte(data), writeFilePermission)

}

func getValue(idx int, file *ast.File) int64 {
	value, err := strconv.Atoi(file.Decls[idx].(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Values[0].(*ast.BasicLit).Value)
	if err != nil {
		panic(err)
	}
	return int64(value)
}
