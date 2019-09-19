package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/dylanlott/exchange/db"
	"github.com/dylanlott/exchange/server"
)

func main() {
	log.Printf("connecting to %s", connString())
	d, err := db.Open("postgres", connString())
	if err != nil {
		log.Fatalf("failed to start database %+v", err)
		return
	}

	// ping the DB to check connection is working
	err = d.Ping()
	if err != nil {
		log.Fatalf("failed to open connection to database %+v", err)
		return
	}

	handler := server.NewRouter(d)
	srv := &http.Server{
		Addr:         ":9000",
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("starting server %+v", srv)
	go srv.ListenAndServe()

	// Wait for an interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	// Attempt a graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func connString() string {
	return fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		"127.0.0.1", 5432, "postgres", "", "exchange")
}
