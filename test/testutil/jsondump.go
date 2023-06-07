package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type jsonOption struct {
	bodyOpts []cmp.Option
	bodyFn   func([]byte)
}

func NewJSONOption(opts ...JSONOption) *jsonOption {
	j := &jsonOption{}
	for _, opt := range opts {
		switch o := opt.(type) {
		case *InspectBodyOption:
			j.bodyFn = o.fn
		case *IgnoreFieldsOption:
			j.bodyOpts = append(j.bodyOpts, ignoreMapKeys(o.keys...))
		default:
			panic("option not implemented")
		}
	}

	return j
}

func DumpJSONFile(fileName string, v any, opts ...JSONOption) error {
	type dumpAndCompare struct {
		dumper
		comparer
	}

	dnc := dumpAndCompare{
		dumper:   NewJSONDumper(v),
		comparer: NewJSONComparer(opts...),
	}

	return Dump(fileName, dnc)
}

// DumpJSON dumps a type as json.
func DumpJSON(t *testing.T, v any, opts ...JSONOption) {
	t.Helper()

	typeName := strings.Join(typeName(v), "_")
	fileName := filepath.Join("./testdata", t.Name(), typeName)
	fileName = fmt.Sprintf("./%s.json", fileName)

	if err := DumpJSONFile(fileName, v, opts...); err != nil {
		t.Fatal(err)
	}
}

type JSONComparer struct {
	opt *jsonOption
}

func NewJSONComparer(opts ...JSONOption) *JSONComparer {
	return &JSONComparer{
		opt: NewJSONOption(opts...),
	}
}

func (c *JSONComparer) Compare(a, b []byte) error {
	// Get slice of data with optional leading whitespace removed.
	// See RFC 7159, Section 2 for the definition of JSON whitespace.
	a = bytes.TrimLeft(a, " \t\r\n")
	b = bytes.TrimLeft(b, " \t\r\n")

	unmarshal := func(j []byte) (any, error) {
		isObject := len(j) > 0 && j[0] == '{'
		var m any
		if isObject {
			m = make(map[string]any)
		}
		if err := json.Unmarshal(j, &m); err != nil {
			return nil, err
		}
		return m, nil
	}

	want, err := unmarshal(a)
	if err != nil {
		return err
	}

	got, err := unmarshal(b)
	if err != nil {
		return err
	}

	if c.opt.bodyFn != nil {
		c.opt.bodyFn(b)
	}

	return cmpDiff(want, got, c.opt.bodyOpts...)
}

type JSONDumper struct {
	v any
}

func NewJSONDumper(v any) *JSONDumper {
	return &JSONDumper{v: v}
}

func (d *JSONDumper) Dump() ([]byte, error) {
	return marshal(d.v)
}

func marshal(v any) ([]byte, error) {
	// If it is byte, pretty print.
	b, ok := v.([]byte)
	if ok {
		if !json.Valid(b) {
			return b, nil
		}

		var bb bytes.Buffer
		if err := json.Indent(&bb, b, "", " "); err != nil {
			return nil, err
		}

		return bb.Bytes(), nil
	}

	return json.MarshalIndent(v, "", " ")
}
