package api

import (
	"testing"

	"github.com/allyourbase/ayb/internal/schema"
	"github.com/allyourbase/ayb/internal/testutil"
)

func TestGetOrCreateExpand(t *testing.T) {
	rec := map[string]any{"id": 1, "name": "test"}

	// First call creates the expand map.
	expand := getOrCreateExpand(rec)
	testutil.NotNil(t, expand)

	expand["author"] = map[string]any{"id": 10, "name": "Alice"}

	// Second call returns the same map.
	expand2 := getOrCreateExpand(rec)
	testutil.Equal(t, expand2["author"].(map[string]any)["name"], "Alice")
}

func TestGetOrCreateExpandExisting(t *testing.T) {
	existing := map[string]any{"tags": []string{"go"}}
	rec := map[string]any{"id": 1, "expand": existing}

	// Should return the existing expand map.
	expand := getOrCreateExpand(rec)
	testutil.NotNil(t, expand)

	// The existing map should be the same object.
	expand["author"] = "test"
	testutil.Equal(t, existing["author"], "test")
}

func TestFindRelationByFieldName(t *testing.T) {
	tbl := &schema.Table{
		Name:   "posts",
		Schema: "public",
		Relationships: []*schema.Relationship{
			{
				FieldName:   "author",
				Type:        "many-to-one",
				FromColumns: []string{"author_id"},
				ToColumns:   []string{"id"},
				ToSchema:    "public",
				ToTable:     "authors",
			},
		},
	}

	found := findRelation(tbl, "author")
	testutil.NotNil(t, found)
	testutil.Equal(t, found.Type, "many-to-one")
}

func TestFindRelationByColumnName(t *testing.T) {
	tbl := &schema.Table{
		Name:   "posts",
		Schema: "public",
		Relationships: []*schema.Relationship{
			{
				FieldName:   "author",
				Type:        "many-to-one",
				FromColumns: []string{"author_id"},
				ToColumns:   []string{"id"},
				ToSchema:    "public",
				ToTable:     "authors",
			},
		},
	}

	found := findRelation(tbl, "author_id")
	testutil.NotNil(t, found)
	testutil.Equal(t, found.FieldName, "author")
}

func TestFindRelationNotFound(t *testing.T) {
	tbl := &schema.Table{
		Name:   "posts",
		Schema: "public",
		Relationships: []*schema.Relationship{
			{
				FieldName:   "author",
				Type:        "many-to-one",
				FromColumns: []string{"author_id"},
				ToColumns:   []string{"id"},
				ToSchema:    "public",
				ToTable:     "authors",
			},
		},
	}

	found := findRelation(tbl, "nonexistent")
	testutil.True(t, found == nil, "expected nil for nonexistent relation")
}

func TestFindRelationColumnNameOnlyMatchesManyToOne(t *testing.T) {
	tbl := &schema.Table{
		Name:   "users",
		Schema: "public",
		Relationships: []*schema.Relationship{
			{
				FieldName:   "posts",
				Type:        "one-to-many",
				FromColumns: []string{"id"},
				ToColumns:   []string{"author_id"},
				ToSchema:    "public",
				ToTable:     "posts",
			},
		},
	}

	// Column name fallback should NOT match one-to-many relationships.
	found := findRelation(tbl, "id")
	testutil.True(t, found == nil, "column name fallback should not match one-to-many")

	// But field name still matches.
	found = findRelation(tbl, "posts")
	testutil.NotNil(t, found)
	testutil.Equal(t, found.Type, "one-to-many")
}

func TestCountKnownColumns(t *testing.T) {
	tbl := &schema.Table{
		Columns: []*schema.Column{
			{Name: "id"},
			{Name: "name"},
			{Name: "email"},
		},
	}

	tests := []struct {
		name string
		data map[string]any
		want int
	}{
		{"all known", map[string]any{"name": "a", "email": "b"}, 2},
		{"some known", map[string]any{"name": "a", "fake": "b"}, 1},
		{"none known", map[string]any{"fake1": "a", "fake2": "b"}, 0},
		{"empty", map[string]any{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countKnownColumns(tbl, tt.data)
			testutil.Equal(t, got, tt.want)
		})
	}
}
