package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var (
	listenAddress = os.Getenv("LISTEN_ADDRESS")
	adminToken    = os.Getenv("ADMIN_TOKEN")
)

func run() error {
	ln, err := net.Listen("tcp", listenAddress)
	if err != nil {
		panic(err)
	}
	log.Printf("listening on http://%v", ln.Addr())

	s := &http.Server{
		Handler: liveServer{},
	}
	errc := make(chan error, 1)
	go func() {
		errc <- s.Serve(ln)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return s.Shutdown(ctx)
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)

	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
