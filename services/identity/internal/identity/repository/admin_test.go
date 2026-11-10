package repository

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"identity/internal/identity/model"
)

// DryRun checks the generated PostgreSQL queries without connecting to a database.
func TestAdminQueries(t *testing.T) {
	db, err := gorm.Open(postgres.Open("host=synthetic.invalid dbname=synthetic"), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true, Logger: logger.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pool.Close(); err != nil {
			t.Error(err)
		}
	})
	var queries []string
	var variables [][]any
	err = db.Callback().Query().After("gorm:query").Register("test:capture_admin_sql", func(tx *gorm.DB) {
		queries = append(queries, tx.Statement.SQL.String())
		variables = append(variables, append([]any(nil), tx.Statement.Vars...))
	})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewUserRepository(db)
	if _, _, err := repo.ListAdmins(context.Background(), 2, 4, "", ""); err != nil {
		t.Fatal(err)
	}
	if len(queries) != 2 || !strings.Contains(queries[0], "count(*)") || !strings.Contains(queries[1], "ORDER BY created_at DESC, id DESC") {
		t.Fatal("list must count and read with deterministic ordering")
	}
	for i, query := range queries {
		if !strings.Contains(query, "role = $1") || variables[i][0] != model.RoleAdmin {
			t.Fatal("both queries must filter ADMIN")
		}
	}
	if !strings.Contains(queries[1], "LIMIT $2 OFFSET $3") || variables[1][1] != 2 || variables[1][2] != 4 {
		t.Fatal("pagination must reach the SQL query")
	}
	queries, variables = nil, nil
	id := uuid.New()
	if _, err := repo.FindAdminByID(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if len(queries) != 1 || !strings.Contains(queries[0], "id = $1 AND role = $2") || variables[0][0] != id || variables[0][1] != model.RoleAdmin {
		t.Fatal("detail must filter both ID and ADMIN role")
	}

	for _, tt := range []struct {
		name, search, status, pattern string
	}{
		{name: "search only", search: "AnDi", pattern: "%AnDi%"},
		{name: "active only", status: model.StatusActive},
		{name: "inactive only", status: model.StatusInactive},
		{name: "search and status", search: "admin@example.invalid", status: model.StatusInactive, pattern: "%admin@example.invalid%"},
		{name: "literal wildcard characters", search: `50%_\`, pattern: `%50\%\_\\%`},
		{name: "SQL input stays a parameter", search: "' OR true --", pattern: "%' OR true --%"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			queries, variables = nil, nil
			if _, _, err := repo.ListAdmins(context.Background(), 20, 0, tt.search, tt.status); err != nil {
				t.Fatal(err)
			}
			if len(queries) != 2 {
				t.Fatal("expected count and row query")
			}
			expected := []any{model.RoleAdmin}
			if tt.search != "" {
				expected = append(expected, tt.pattern, tt.pattern)
			}
			if tt.status != "" {
				expected = append(expected, tt.status)
			}
			if !reflect.DeepEqual(variables[0], expected) || !reflect.DeepEqual(variables[1][:len(expected)], expected) {
				t.Fatal("count and rows must receive the same bound filters")
			}
			for _, query := range queries {
				if !strings.Contains(query, "role = $1") {
					t.Fatal("ADMIN filter must always apply")
				}
				if tt.search != "" && !strings.Contains(query, "AND ((full_name ILIKE $2 OR email ILIKE $3))") {
					t.Fatalf("search must group name/email conditions: %s", query)
				}
				if tt.status != "" && !strings.Contains(query, "AND status = $") {
					t.Fatal("status must combine with AND")
				}
				if tt.search != "" && strings.Contains(query, tt.search) {
					t.Fatal("search input must stay out of SQL text")
				}
			}
		})
	}
}
