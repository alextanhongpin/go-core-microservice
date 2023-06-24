package testutil

import (
	"encoding/json"
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
	"github.com/mjibson/sqlfmt"
	pg_query "github.com/pganalyze/pg_query_go/v4"
)

type PostgresSQLDumper struct {
	dump *SQLDump
	opts *sqlOption
}

func NewPostgresSQLDumper(dump *SQLDump, opts ...SQLOption) *PostgresSQLDumper {
	return &PostgresSQLDumper{
		dump: dump,
		opts: NewSQLOption(opts...),
	}
}

func (d *PostgresSQLDumper) Dump() ([]byte, error) {
	result, err := pg_query.Parse(d.dump.Stmt)
	if err != nil {
		return nil, err
	}

	query, err := pg_query.Deparse(result)
	if err != nil {
		return nil, err
	}

	queryNorm := query

	args := make(map[string]any)

	if d.opts.normalize {
		queryNorm, args, err = normalizePostgres(query)
		if err != nil {
			return nil, err
		}
	}

	queryNormPretty, err := sqlfmt.FmtSQL(tree.PrettyCfg{
		LineWidth: dynamicLineWidth(queryNorm),
		TabWidth:  2,
		JSONFmt:   true,
	}, []string{queryNorm})
	if err != nil {
		return nil, err
	}

	queryPretty, err := sqlfmt.FmtSQL(tree.PrettyCfg{
		LineWidth: dynamicLineWidth(query),
		TabWidth:  2,
		JSONFmt:   true,
	}, []string{query})
	if err != nil {
		return nil, err
	}

	for k, v := range toArgsMap(d.dump.Args) {
		args[k] = v
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
