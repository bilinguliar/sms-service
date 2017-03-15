package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	mb "github.com/messagebird/go-rest-api"
)

const (
	smsEndpoint = "/messages"
	sendRate    = 1000 * time.Millisecond // SMS send rate.
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	var (
		port     string
		token    string
		queueLen int
	)

	// Not doing this in init to avoid possibility to use flag vars directly.
	// Want them to be passed to consumers.
	flag.StringVar(&port, "port", "8080", "Specifies port that server will use to accept connections")
	flag.StringVar(&token, "token", "", "SMS Gateway API token")
	flag.IntVar(&queueLen, "queue_length", 1000, "SMS Queue size. Rate limiting is enabled: 1 SMS/s.")

	flag.Parse()

	client := NewMsgBirdClient(
		mb.New(token),
		queueLen,
		sendRate,
	)

	mux := http.NewServeMux()

	handler := NewHandler(client)
	mux.HandleFunc(smsEndpoint, handler.HandleMsg)

	// Enabling timeouts as requests will be blocked if SMS queue is full.
	// https://blog.cloudflare.com/exposing-go-on-the-internet/
	srv := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 2 * time.Second,
		IdleTimeout:  4 * time.Second, // Requires Go 1.8
		Handler:      mux,
	}

	// TODO handle graceful shutdown.
	log.Fatal(srv.ListenAndServe())
}
