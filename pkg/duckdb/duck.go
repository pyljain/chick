package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

type DB struct {
	client *sql.DB
}

// /path/to/foo.db?access_mode=read_only&threads=4
func NewDB(connectionString, dataFilesPath string) (*DB, error) {
	db, err := sql.Open("duckdb", connectionString)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("CREATE TABLE data AS SELECT * FROM '%s/*.parquet'", dataFilesPath)
	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	return &DB{
		client: db,
	}, nil
}

func (d *DB) Query(ctx context.Context, query string) ([]map[string]any, error) {
	res, err := d.client.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if err = res.Err(); err != nil {
		return nil, err
	}

	// 1. Get the column names dynamically
	columns, err := res.Columns()
	if err != nil {
		return nil, err
	}

	// This slice will hold our generic list of records
	var records []map[string]any

	for res.Next() {
		// Create a slice to hold the raw values for this row
		values := make([]any, len(columns))

		// Create a slice of pointers pointing to the values slice
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// 3. Scan the data into the pointers
		if err := res.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// 4. Map column names to their fetched values
		rowMap := make(map[string]any)
		for i, colName := range columns {
			val := values[i]

			// DuckDB may return byte slices for strings, handle if needed
			if b, ok := val.([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}

		records = append(records, rowMap)
	}

	return records, nil

}
