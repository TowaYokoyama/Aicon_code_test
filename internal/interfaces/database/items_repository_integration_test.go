//go:build integration

package database

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	domainErrors "Aicon-assignment/internal/domain/errors"
	"Aicon-assignment/internal/domain/entity"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSqlHandler struct {
	db *sql.DB
}

func (h *testSqlHandler) Execute(ctx context.Context, statement string, args ...interface{}) (Result, error) {
	result, err := h.db.ExecContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	return &testResult{result: result}, nil
}

func (h *testSqlHandler) Query(ctx context.Context, statement string, args ...interface{}) (Rows, error) {
	rows, err := h.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	return &testRows{rows: rows}, nil
}

func (h *testSqlHandler) QueryRow(ctx context.Context, statement string, args ...interface{}) Row {
	return &testRow{row: h.db.QueryRowContext(ctx, statement, args...)}
}

func (h *testSqlHandler) Close() error {
	return h.db.Close()
}

type testResult struct {
	result sql.Result
}

func (r *testResult) LastInsertId() (int64, error) {
	return r.result.LastInsertId()
}

func (r *testResult) RowsAffected() (int64, error) {
	return r.result.RowsAffected()
}

type testRows struct {
	rows *sql.Rows
}

func (r *testRows) Next() bool {
	return r.rows.Next()
}

func (r *testRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *testRows) Close() error {
	return r.rows.Close()
}

func (r *testRows) Err() error {
	return r.rows.Err()
}

type testRow struct {
	row *sql.Row
}

func (r *testRow) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN is not set")
	}

	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func resetItemsTable(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), "TRUNCATE TABLE items")
	require.NoError(t, err)
}

func TestItemRepository_CreateAndFindByID(t *testing.T) {
	db := openTestDB(t)
	resetItemsTable(t, db)

	repo := &ItemRepository{SqlHandler: &testSqlHandler{db: db}}
	ctx := context.Background()

	item, err := entity.NewItem("テスト", "時計", "ROLEX", 1000, "2023-01-01")
	require.NoError(t, err)

	created, err := repo.Create(ctx, item)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.NotZero(t, created.ID)

	fetched, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "テスト", fetched.Name)
	assert.Equal(t, "時計", fetched.Category)
	assert.Equal(t, "ROLEX", fetched.Brand)
	assert.Equal(t, 1000, fetched.PurchasePrice)
	assert.Equal(t, "2023-01-01", fetched.PurchaseDate)
}

func TestItemRepository_Update(t *testing.T) {
	db := openTestDB(t)
	resetItemsTable(t, db)

	repo := &ItemRepository{SqlHandler: &testSqlHandler{db: db}}
	ctx := context.Background()

	item, err := entity.NewItem("元の名前", "時計", "ROLEX", 1000, "2023-01-01")
	require.NoError(t, err)

	created, err := repo.Create(ctx, item)
	require.NoError(t, err)

	created.Name = "更新後"
	created.Brand = "CHANEL"
	created.PurchasePrice = 2000
	created.UpdatedAt = time.Now()

	require.NoError(t, repo.Update(ctx, created))

	fetched, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "更新後", fetched.Name)
	assert.Equal(t, "CHANEL", fetched.Brand)
	assert.Equal(t, 2000, fetched.PurchasePrice)
}

func TestItemRepository_Delete(t *testing.T) {
	db := openTestDB(t)
	resetItemsTable(t, db)

	repo := &ItemRepository{SqlHandler: &testSqlHandler{db: db}}
	ctx := context.Background()

	item, err := entity.NewItem("テスト", "時計", "ROLEX", 1000, "2023-01-01")
	require.NoError(t, err)

	created, err := repo.Create(ctx, item)
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, created.ID))

	_, err = repo.FindByID(ctx, created.ID)
	assert.ErrorIs(t, err, domainErrors.ErrItemNotFound)
}

func TestItemRepository_GetSummaryByCategory(t *testing.T) {
	db := openTestDB(t)
	resetItemsTable(t, db)

	repo := &ItemRepository{SqlHandler: &testSqlHandler{db: db}}
	ctx := context.Background()

	items := []struct {
		name     string
		category string
	}{
		{"A", "時計"},
		{"B", "時計"},
		{"C", "バッグ"},
	}

	for _, it := range items {
		item, err := entity.NewItem(it.name, it.category, "BRAND", 1000, "2023-01-01")
		require.NoError(t, err)
		_, err = repo.Create(ctx, item)
		require.NoError(t, err)
	}

	summary, err := repo.GetSummaryByCategory(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, summary["時計"])
	assert.Equal(t, 1, summary["バッグ"])
}
