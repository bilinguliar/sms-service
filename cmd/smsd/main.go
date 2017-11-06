package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	smsd "github.com/cooldarkdryplace/sms-service"

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

	client := smsd.NewMsgBirdClient(
		mb.New(token),
		queueLen,
		sendRate,
	)

	mux := http.NewServeMux()

	handler := smsd.NewHandler(client)
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

	errChan := make(chan error)
	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, os.Interrupt)

	go func() {
		errChan <- srv.ListenAndServe()
	}()

	log.Printf("SMSd started %s\n", time.Now().UTC())

	select {
	case err := <-errChan:
		log.Fatal(err)
	case <-signalChan:
		log.Println("Interrupt recieved. Graceful shutdown.")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Server shutdown failed with error: %s\n", err)
		}
	}
}
