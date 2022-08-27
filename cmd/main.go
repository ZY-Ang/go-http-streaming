package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"go-http-streaming/pkg/contextutil"
)

// testEndpoint is a simple example of how to keep writing to a file in a for loop
func testEndpoint(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got /test request\n")

	// Write headers so browser will know this is a file to be downloaded
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.txt\"", "test"))

	// iterate over something arbitrary, like a dataset or do some business logic
	for i := 0; i < 1000000; i++ {
		_, err := w.Write([]byte(fmt.Sprintf("%d\n", i+1)))
		if err != nil {
			fmt.Printf("failed to write at %d: %v\n", i, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	fmt.Println("done /test request")
}

// sqlEndpoint is an example of how to keep writing
func sqlEndpoint(ctx context.Context, db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("got / request\n")
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.csv\"", "test"))

		// write CSV headers
		_, err := w.Write([]byte("id,some_col\n"))
		if err != nil {
			fmt.Println("failed to write headers")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// paginate db
		minId := uint64(0)
		const pageSize = 128
		pageCounterForLogging := 0
		for {
			pageCounterForLogging++
			fmt.Printf("iterating over page %d", pageCounterForLogging)
			rows, err := db.QueryContext(
				ctx,
				`
					SELECT id, some_col
					FROM test_tab
					WHERE id > ?
					LIMIT ?;
					`,
				minId,
				pageSize,
			)
			if err != nil {
				fmt.Printf("failed to query db at %d: %v\n", minId, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			batchCount := 0
			sb := strings.Builder{}
			for rows.Next() {
				batchCount++
				var id uint64
				var someCol string
				err := rows.Scan(&id, &someCol)
				if err != nil {
					fmt.Printf("failed to scan at %d: %v\n", minId, err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				minId = id
				sb.WriteString(fmt.Sprintf("%d,%s\n", id, someCol))
			}
			_ = rows.Close()
			if batchCount == 0 {
				// nothing to write
				_ = sb.String()
				break
			}

			_, err = w.Write([]byte(sb.String()))
			if err != nil {
				fmt.Printf("failed to write rows at %d: %v\n", minId, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if batchCount < pageSize {
				// no more next page
				break
			}
		}

		w.WriteHeader(http.StatusOK)
		fmt.Println("done / request")
	}
}

func main() {
	ctx := contextutil.WithShutdown(context.Background())
	fmt.Println("starting server")

	serveDbEndpoint := true
	db, err := setupDb(ctx)
	if err != nil {
		fmt.Printf("error setting up db: %v\n", err)
		serveDbEndpoint = false
	}
	defer func() {
		if serveDbEndpoint {
			_ = db.Close()
		}
	}()

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: nil, // use default mux for ease
	}
	http.HandleFunc("/test", testEndpoint)
	if serveDbEndpoint {
		http.HandleFunc("/", sqlEndpoint(ctx, db))
	}

	serveErrChan := make(chan error, 1)
	defer close(serveErrChan)
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil {
			serveErrChan <- err
		}
	}()
	defer func() {
		err := httpServer.Shutdown(ctx)
		if err != nil {
			fmt.Printf("error shutting down http server: %v\n", err)
		}
	}()

	fmt.Println("started server")
	select {
	case err := <-serveErrChan:
		fmt.Printf("error starting server: %v\n", err)
	case <-ctx.Done():
		fmt.Println("stopping server")
	}
}
