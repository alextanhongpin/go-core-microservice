package sqldump

import (
	"fmt"

	"github.com/alextanhongpin/core/storage/sql/sqlformat"
	querypb "vitess.io/vitess/go/vt/proto/query"

	"vitess.io/vitess/go/vt/sqlparser"
)

func DumpMySQL(sql *SQL, marshalFunc func(v any) ([]byte, error)) ([]byte, error) {
	q, err := standardizeMySQL(sql.Query)
	if err != nil {
		return nil, err
	}

	q, err = sqlformat.Format(q)
	if err != nil {
		return nil, err
	}

	args := make(map[string]any)
	for i, v := range sql.Args {
		// sqlparser replaces all '?' with ':v1', ':v2', ':vn'
		// ...
		k := fmt.Sprintf(":v%d", i+1)
		args[k] = v
	}

	a, err := marshalFunc(args)
	if err != nil {
		return nil, err
	}

	b, err := marshalFunc(sql.Result)
	if err != nil {
		return nil, err
	}

	return []byte(dump(q, a, b)), nil
}

// MatchMySQLQuery checks if two queries are equal,
// ignoring variables.
func MatchMySQLQuery(a, b string) (bool, error) {
	x, err := normalizeMySQL(a)
	if err != nil {
		return false, err
	}

	y, err := normalizeMySQL(b)
	if err != nil {
		return false, err
	}

	return x == y, nil
}

func standardizeMySQL(q string) (string, error) {
	stmt, err := sqlparser.Parse(q)
	if err != nil {
		return "", err
	}

	q = sqlparser.String(stmt)

	// sqlparser replaces all ? with the format :v1, :v2,
	// :vn ...
	return q, nil
}

// Referred from sqlparser.QueryMatchesTemplates(q, []string{q})
func normalizeMySQL(q string) (string, error) {
	bv := make(map[string]*querypb.BindVariable)
	q, err := sqlparser.NormalizeAlphabetically(q)
	if err != nil {
		return "", err
	}

	stmt, reservedVars, err := sqlparser.Parse2(q)
	if err != nil {
		return "", err
	}

	err = sqlparser.Normalize(stmt, sqlparser.NewReservedVars("", reservedVars), bv)
	if err != nil {
		return "", err
	}

	normalized := sqlparser.CanonicalString(stmt)

	return normalized, nil
}
