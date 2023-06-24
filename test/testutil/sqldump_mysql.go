package testutil

import (
	"encoding/json"
	"fmt"
	"strings"

	querypb "vitess.io/vitess/go/vt/proto/query"
	"vitess.io/vitess/go/vt/sqlparser"
)

type MySQLDumper struct {
	dump *SQLDump
	opts *sqlOption
}

func NewMySQLDumper(dump *SQLDump, opts ...SQLOption) *MySQLDumper {
	return &MySQLDumper{
		dump: dump,
		opts: NewSQLOption(opts...),
	}
}

func (d *MySQLDumper) Dump() ([]byte, error) {
	stmt, known, err := sqlparser.Parse2(d.dump.Stmt)
	if err != nil {
		return nil, err
	}

	query := sqlparser.String(stmt)
	queryPretty := query

	args := make(map[string]any)
	for i, v := range d.dump.Args {
		key := fmt.Sprintf("v%d", i+1)
		args[key] = v
	}

	if d.opts.normalize {
		bv := make(map[string]*querypb.BindVariable)
		err = sqlparser.Normalize(stmt, sqlparser.NewReservedVars("bv", known), bv)
		if err != nil {
			return nil, err
		}

		for k, v := range bv {
			if _, ok := args[k]; ok {
				continue
			}

			if b := v.GetValue(); len(b) > 0 {
				args[k] = string(b)
			} else {
				vals := make([]string, len(v.GetValues()))
				for i, v := range v.GetValues() {
					vals[i] = string(v.GetValue())
				}
				args[k] = vals
			}
		}
	}

	// Unfortunately the prettier doesn't work with ":".
	queryNorm := sqlparser.String(stmt)
	queryNormPretty := queryNorm
	if isPythonInstalled {
		normBytes, err := sqlformat(queryNorm)
		if err == nil {
			queryNormPretty = string(normBytes)
		}
		bytes, err := sqlformat(query)
		if err == nil {
			queryPretty = string(bytes)
		}
	}

	argsBytes, err := json.MarshalIndent(args, "", " ")
	if err != nil {
		return nil, err
	}

	rows, err := json.MarshalIndent(d.dump.Rows, "", " ")
	if err != nil {
		return nil, err
	}

	lineBreak := string(LineBreak)
	querySection := []string{
		queryStmtSection,
		queryPretty,
		lineBreak,
	}

	queryNormalizedSection := []string{
		queryNormalizedStmtSection,
		queryNormPretty,
		lineBreak,
	}

	argsSection := []string{
		argsStmtSection,
		string(argsBytes),
		lineBreak,
	}

	rowsSection := []string{
		rowsStmtSection,
		string(rows),
	}

	res := append([]string{}, querySection...)
	if d.opts.normalize {
		res = append(res, queryNormalizedSection...)
	}
	res = append(res, argsSection...)
	res = append(res, rowsSection...)

	return []byte(strings.Join(res, string(LineBreak))), nil
}
