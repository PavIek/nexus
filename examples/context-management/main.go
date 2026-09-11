package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func handler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	select {
	case <-time.After(5 * time.Second):
		fmt.Fprintln(w, "Response after delay")
	case <-ctx.Done():
		log.Println("Client disconnected")
	}

	db, err := sql.Open("driver-name", "database=test1")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, "SELECT * FROM users")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
}

func main() {
	ctx := context.WithValue(context.Background(), "requestId", "12345")
	handleRequest(ctx)

	ctx, cancel := context.WithCancel(context.Background())

	for i := 0; i < 10; i++ {
		go worker(ctx, i)
	}

	cancel()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	<-ctx.Done()
	fmt.Println("Shutting down...")
}

func worker(ctx context.Context, i int) {}

func handleRequest(ctx context.Context) {
	log.Printf("Handling request with ID: %v", ctx.Value("requestID"))
}

type Data struct{}

func streamData(ctx context.Context, ch <-chan Data) {
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-ch:
			process(data)
		}
	}
}

func process(data Data) {}

func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rateLimited := true

		ctx := context.WithValue(r.Context(), "rateLimited", rateLimited)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func handler2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if limited, ok := ctx.Value("rateLimited").(bool); ok && limited {
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		return
	}
	fmt.Fprintln(w, "Request accepted")
}
