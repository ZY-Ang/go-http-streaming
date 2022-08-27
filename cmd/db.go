package main

import (
	"context"
	"database/sql"
	"fmt"
)

func setupDb(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open(
		"mysql",
		"",
	)
	if err != nil {
		fmt.Println("error opening db")
		return nil, err
	}
	shouldSeedData := false
	row := db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM test_tab;
	`)
	err = row.Err()
	if err != nil {
		fmt.Println("error counting db")
		return nil, err
	}
}
