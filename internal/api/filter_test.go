package api

import (
	"strings"
	"testing"

	"github.com/allyourbase/ayb/internal/schema"
	"github.com/allyourbase/ayb/internal/testutil"
)

func filterTestTable() *schema.Table {
	return &schema.Table{
		Schema: "public",
		Name:   "users",
		Kind:   "table",
		Columns: []*schema.Column{
			{Name: "id", Position: 1, TypeName: "integer", IsPrimaryKey: true},
			{Name: "name", Position: 2, TypeName: "text"},
			{Name: "email", Position: 3, TypeName: "varchar"},
			{Name: "age", Position: 4, TypeName: "integer"},
			{Name: "status", Position: 5, TypeName: "text"},
			{Name: "active", Position: 6, TypeName: "boolean"},
		},
		PrimaryKey: []string{"id"},
	}
}

// --- Tokenizer tests ---

func TestTokenizeSimple(t *testing.T) {
	tokens, err := tokenize("name='Alice'")
	testutil.NoError(t, err)
	testutil.SliceLen(t, tokens, 3)
	testutil.Equal(t, tokens[0].kind, tokIdent)
	testutil.Equal(t, tokens[0].value, "name")
	testutil.Equal(t, tokens[1].kind, tokOp)
	testutil.Equal(t, tokens[1].value, "=")
	testutil.Equal(t, tokens[2].kind, tokString)
	testutil.Equal(t, tokens[2].value, "Alice")
}

func TestTokenizeWithSpaces(t *testing.T) {
	tokens, err := tokenize("age > 25")
	testutil.NoError(t, err)
	testutil.SliceLen(t, tokens, 3)
	testutil.Equal(t, tokens[0].value, "age")
	testutil.Equal(t, tokens[1].value, ">")
	testutil.Equal(t, tokens[2].value, "25")
	testutil.Equal(t, tokens[2].kind, tokNumber)
}

func TestTokenizeAnd(t *testing.T) {
	tokens, err := tokenize("a=1 && b=2")
	testutil.NoError(t, err)
	testutil.SliceLen(t, tokens, 7)
	testutil.Equal(t, tokens[3].kind, tokAnd)
}

func TestTokenizeOr(t *testing.T) {
	tokens, err := tokenize("a=1 || b=2")
	testutil.NoError(t, err)
	testutil.SliceLen(t, tokens, 7)
	testutil.Equal(t, tokens[3].kind, tokOr)
}

func TestTokenizeAndKeyword(t *testing.T) {
	tokens, err := tokenize("a=1 AND b=2")
	testutil.NoError(t, err)
	testutil.SliceLen(t, tokens, 7)
	testutil.Equal(t, tokens[3].kind, tokAnd)
}

func TestTokenizeOrKeyword(t *testing.T) {
	tokens, err := tokenize("a=1 OR b=2")
	testutil.NoError(t, err)
	testutil.SliceLen(t, tokens, 7)
	testutil.Equal(t, tokens[3].kind, tokOr)
}

func TestTokenizeParens(t *testing.T) {
	tokens, err := tokenize("(a=1)")
	testutil.NoError(t, err)
	testutil.SliceLen(t, tokens, 5)
	testutil.Equal(t, tokens[0].kind, tokLParen)
	testutil.Equal(t, tokens[4].kind, tokRParen)
}

func TestTokenizeBool(t *testing.T) {
	tokens, err := tokenize("active=true")
	testutil.NoError(t, err)
	testutil.SliceLen(t, tokens, 3)
	testutil.Equal(t, tokens[2].kind, tokBool)
	testutil.Equal(t, tokens[2].value, "true")
}

func TestTokenizeNull(t *testing.T) {
	tokens, err := tokenize("name=null")
	testutil.NoError(t, err)
	testutil.SliceLen(t, tokens, 3)
	testutil.Equal(t, tokens[2].kind, tokNull)
}

func TestTokenizeIn(t *testing.T) {
	tokens, err := tokenize("status IN ('a','b','c')")
	testutil.NoError(t, err)
	testutil.Equal(t, tokens[1].kind, tokIn)
	testutil.Equal(t, tokens[2].kind, tokLParen)
}

func TestTokenizeOperators(t *testing.T) {
	tests := []struct {
		input string
		op    string
	}{
		{"a=1", "="},
		{"a!=1", "!="},
		{"a>1", ">"},
		{"a>=1", ">="},
		{"a<1", "<"},
		{"a<=1", "<="},
		{"a~'x'", "~"},
		{"a!~'x'", "!~"},
	}

	for _, tc := range tests {
		tokens, err := tokenize(tc.input)
		testutil.NoError(t, err)
		testutil.Equal(t, tokens[1].kind, tokOp)
		testutil.Equal(t, tokens[1].value, tc.op)
	}
}

func TestTokenizeFloat(t *testing.T) {
	tokens, err := tokenize("age>3.14")
	testutil.NoError(t, err)
	testutil.Equal(t, tokens[2].value, "3.14")
	testutil.Equal(t, tokens[2].kind, tokNumber)
}

func TestTokenizeEscapedQuote(t *testing.T) {
	tokens, err := tokenize(`name='it\'s'`)
	testutil.NoError(t, err)
	testutil.SliceLen(t, tokens, 3)
	testutil.Equal(t, tokString, tokens[2].kind)
	testutil.Equal(t, "it's", tokens[2].value)
}

func TestTokenizeEscapedBackslash(t *testing.T) {
	tokens, err := tokenize(`name='a\\b'`)
	testutil.NoError(t, err)
	testutil.SliceLen(t, tokens, 3)
	testutil.Equal(t, `a\b`, tokens[2].value)
}

func TestTokenizeUnterminatedString(t *testing.T) {
	_, err := tokenize("name='unterminated")
	testutil.True(t, err != nil, "expected error for unterminated string")
	testutil.Contains(t, err.Error(), "unterminated")
}

func TestTokenizeUnexpectedChar(t *testing.T) {
	_, err := tokenize("name=$1")
	testutil.True(t, err != nil, "expected error for unexpected char")
	testutil.Contains(t, err.Error(), "unexpected")
}

// --- Parser tests ---

func TestParseFilterSimpleEquals(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "name='Alice'")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `"name" = $1`)
	testutil.SliceLen(t, args, 1)
	testutil.Equal(t, args[0].(string), "Alice")
}

func TestParseFilterNumber(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "age>25")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `"age" > $1`)
	testutil.SliceLen(t, args, 1)
	testutil.Equal(t, args[0].(int64), int64(25))
}

func TestParseFilterFloat(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "age>3.14")
	testutil.NoError(t, err)
	testutil.Contains(t, sql, `"age" > $1`)
	testutil.Equal(t, args[0].(float64), 3.14)
}

func TestParseFilterBool(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "active=true")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `"active" = $1`)
	testutil.Equal(t, args[0].(bool), true)
}

func TestParseFilterNull(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "name=null")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `"name" IS NULL`)
	testutil.SliceLen(t, args, 0)
}

func TestParseFilterNotNull(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "name!=null")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `"name" IS NOT NULL`)
	testutil.SliceLen(t, args, 0)
}

func TestParseFilterAnd(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "name='Alice' && age>25")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `("name" = $1 AND "age" > $2)`)
	testutil.SliceLen(t, args, 2)
}

func TestParseFilterOr(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "name='Alice' || name='Bob'")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `("name" = $1 OR "name" = $2)`)
	testutil.SliceLen(t, args, 2)
}

func TestParseFilterAndKeyword(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "name='Alice' AND age>25")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `("name" = $1 AND "age" > $2)`)
	testutil.SliceLen(t, args, 2)
}

func TestParseFilterOrKeyword(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "name='Alice' OR name='Bob'")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `("name" = $1 OR "name" = $2)`)
	testutil.SliceLen(t, args, 2)
}

func TestParseFilterParens(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "(name='Alice' || name='Bob') && age>25")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `(("name" = $1 OR "name" = $2) AND "age" > $3)`)
	testutil.SliceLen(t, args, 3)
}

func TestParseFilterLike(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "name~'%Ali%'")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `"name" LIKE $1`)
	testutil.Equal(t, args[0].(string), "%Ali%")
}

func TestParseFilterNotLike(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "name!~'%Ali%'")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `"name" NOT LIKE $1`)
	testutil.Equal(t, args[0].(string), "%Ali%")
}

func TestParseFilterIn(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "status IN ('active','inactive')")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `"status" IN ($1, $2)`)
	testutil.SliceLen(t, args, 2)
	testutil.Equal(t, args[0].(string), "active")
	testutil.Equal(t, args[1].(string), "inactive")
}

func TestParseFilterComplex(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "status='active' && (age>=18 || name='admin')")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `("status" = $1 AND ("age" >= $2 OR "name" = $3))`)
	testutil.SliceLen(t, args, 3)
}

func TestParseFilterEscapedQuote(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, `name='it\'s'`)
	testutil.NoError(t, err)
	testutil.Equal(t, `"name" = $1`, sql)
	testutil.SliceLen(t, args, 1)
	testutil.Equal(t, "it's", args[0].(string))
}

func TestParseFilterUnknownColumn(t *testing.T) {
	tbl := filterTestTable()
	_, _, err := parseFilter(tbl, "nonexistent='x'")
	testutil.True(t, err != nil, "expected error for unknown column")
	testutil.Contains(t, err.Error(), "unknown column")
}

func TestParseFilterEmpty(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, "")
	testutil.True(t, args == nil, "expected nil args")
}

func TestParseFilterMissingOperator(t *testing.T) {
	tbl := filterTestTable()
	_, _, err := parseFilter(tbl, "name 'Alice'")
	testutil.True(t, err != nil, "expected error for missing operator")
	testutil.Contains(t, err.Error(), "expected")
}

func TestParseFilterUnclosedParen(t *testing.T) {
	tbl := filterTestTable()
	_, _, err := parseFilter(tbl, "(name='Alice'")
	testutil.True(t, err != nil, "expected error for unclosed paren")
	testutil.Contains(t, err.Error(), "closing parenthesis")
}

func TestParseFilterMultipleAnd(t *testing.T) {
	tbl := filterTestTable()
	sql, args, err := parseFilter(tbl, "name='A' && age>1 && active=true")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `(("name" = $1 AND "age" > $2) AND "active" = $3)`)
	testutil.SliceLen(t, args, 3)
}

func TestParseFilterOperatorPrecedence(t *testing.T) {
	tbl := filterTestTable()
	// OR has lower precedence than AND, so "a || b && c" should be "a || (b && c)"
	// But our grammar is: or_expr = and_expr (OR and_expr)*
	// So: a=1 || b=2 && c=3 → (a=1) OR ((b=2) AND (c=3))
	sql, args, err := parseFilter(tbl, "name='a' || age>2 && active=true")
	testutil.NoError(t, err)
	testutil.Equal(t, sql, `("name" = $1 OR ("age" > $2 AND "active" = $3))`)
	testutil.SliceLen(t, args, 3)
}

// --- parseSortSQL tests ---

func TestParseSortSQLEmpty(t *testing.T) {
	tbl := filterTestTable()
	testutil.Equal(t, parseSortSQL(tbl, ""), "")
}

func TestParseSortSQLSingleAsc(t *testing.T) {
	tbl := filterTestTable()
	testutil.Equal(t, parseSortSQL(tbl, "name"), `"name" ASC`)
}

func TestParseSortSQLSingleDesc(t *testing.T) {
	tbl := filterTestTable()
	testutil.Equal(t, parseSortSQL(tbl, "-name"), `"name" DESC`)
}

func TestParseSortSQLExplicitAsc(t *testing.T) {
	tbl := filterTestTable()
	testutil.Equal(t, parseSortSQL(tbl, "+name"), `"name" ASC`)
}

func TestParseSortSQLMultiple(t *testing.T) {
	tbl := filterTestTable()
	testutil.Equal(t, parseSortSQL(tbl, "-age,+name"), `"age" DESC, "name" ASC`)
}

func TestParseSortSQLIgnoresInvalidColumns(t *testing.T) {
	tbl := filterTestTable()
	testutil.Equal(t, parseSortSQL(tbl, "-nonexistent,name"), `"name" ASC`)
}

// --- Filter depth limit ---

func TestFilterDepthLimit(t *testing.T) {
	tbl := filterTestTable()
	// Build deeply nested expression: (((((...(id=1)...))))
	nested := strings.Repeat("(", maxFilterDepth+1) + "id=1" + strings.Repeat(")", maxFilterDepth+1)
	_, _, err := parseFilter(tbl, nested)
	testutil.True(t, err != nil, "deeply nested filter should be rejected")
	testutil.ErrorContains(t, err, "too deeply nested")
}

func TestFilterDepthAtLimit(t *testing.T) {
	tbl := filterTestTable()
	// Build expression at exactly the limit — should succeed.
	nested := strings.Repeat("(", maxFilterDepth) + "id=1" + strings.Repeat(")", maxFilterDepth)
	sql, _, err := parseFilter(tbl, nested)
	testutil.NoError(t, err)
	testutil.True(t, len(sql) > 0, "filter at depth limit should produce SQL")
}
