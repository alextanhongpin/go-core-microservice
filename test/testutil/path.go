package testutil

import (
	"path/filepath"
	"strings"
)

const (
	TestData = "testdata"

	ExtJSON = ".json"
	ExtHTTP = ".http"
	ExtSQL  = ".sql"
)

type Path struct {
	TestDir  string
	FilePath string
	FileName string
	FileExt  string
}

func (o *Path) String() string {
	if len(o.FileName) == 0 {
		return filepath.Join(
			o.TestDir,
			o.FilePath+o.FileExt,
		)
	}

	// Get the file extension.
	fileName := string(o.FileName)
	fileExt := filepath.Ext(fileName)
	if fileExt != o.FileExt {
		fileName = fileName + o.FileExt
	}

	return filepath.Join(
		o.TestDir,
		o.FilePath,
		fileName,
	)
}

func NewJSONPath(opts ...JSONOption) *Path {
	opt := &Path{
		TestDir:  TestData,
		FilePath: "",
		FileName: "",
		FileExt:  ExtJSON,
	}

	for _, o := range opts {
		switch v := o.(type) {
		case FilePath:
			opt.FilePath = strings.TrimSuffix(string(v), "/")
		case FileName:
			opt.FileName = string(v)
		}
	}

	return opt
}

func NewHTTPPath(opts ...HTTPOption) *Path {
	opt := &Path{
		TestDir:  TestData,
		FilePath: "",
		FileName: "",
		FileExt:  ExtHTTP,
	}

	for _, o := range opts {
		switch v := o.(type) {
		case FilePath:
			opt.FilePath = strings.TrimSuffix(string(v), "/")
		case FileName:
			opt.FileName = string(v)
		}
	}

	return opt
}

func NewSQLPath(opts ...SQLOption) *Path {
	opt := &Path{
		TestDir:  TestData,
		FilePath: "",
		FileName: "",
		FileExt:  ExtSQL,
	}

	for _, o := range opts {
		switch v := o.(type) {
		case FilePath:
			opt.FilePath = strings.TrimSuffix(string(v), "/")
		case FileName:
			opt.FileName = string(v)
		}
	}

	return opt
}
