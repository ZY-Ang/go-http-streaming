package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

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
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.txt\"", "test"))

		var pingResult string
		err := db.PingContext(ctx)
		if err != nil {
			pingResult = err.Error()
		} else {
			pingResult = "db ping success"
		}
		_, err = w.Write([]byte(fmt.Sprintf("%s\n", pingResult)))
		if err != nil {
			fmt.Printf("failed to write: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
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
		_ = db.Close()
	}

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
