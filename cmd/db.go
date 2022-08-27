package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func setupDb(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open(
		"mysql",
		"mysql:pw123@tcp(localhost:3306)/httpstreampoc",
	)
	if err != nil {
		fmt.Println("error opening db")
		return nil, err
	}
	countRow := db.QueryRowContext(ctx, `
		SELECT COUNT(1) as cnt
		FROM test_tab;
	`)
	err = countRow.Err()
	if err != nil {
		fmt.Println("error counting db")
		return nil, err
	}
	var count uint64
	err = countRow.Scan(&count)
	if err != nil {
		fmt.Println("error scanning count")
		return nil, err
	}
	fmt.Printf("got %d rows in db\n", count)

	return db, nil
}
