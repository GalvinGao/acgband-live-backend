package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/time/rate"
	"nhooyr.io/websocket"
)

var baseTime = time.Now()

type liveServer struct{}

func (s liveServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.URL.Path == "/live" {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		go s.handleWs(c, r)
		return
	}

	if r.URL.Path == "/admin/post" {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+adminToken)) != 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		msg := r.FormValue("msg")
		if msg == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		fanout([]byte("servermsg " + msg))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.NotFound(w, r)
}

func (s liveServer) handleWs(c *websocket.Conn, r *http.Request) {
	l := rate.NewLimiter(rate.Every(time.Millisecond*30), 10)
	go s.writePump(r.Context(), c)
	for {
		err := s.readPump(context.Background(), c, l)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return
		}
		if err != nil {
			fmt.Printf("failed to send to %v: %v", r.RemoteAddr, err)
			return
		}
	}
}

func (s liveServer) readPump(ctx context.Context, c *websocket.Conn, l *rate.Limiter) error {
	err := l.Wait(ctx)
	if err != nil {
		return err
	}

	typ, r, err := c.Read(ctx)
	if err != nil {
		return err
	}
	start := time.Since(baseTime)

	if typ != websocket.MessageText {
		return fmt.Errorf("expected text message")
	}

	op := string(r)
	switch op {
	case "clienthello":
		// echo back server time
		err = c.Write(ctx, websocket.MessageText, []byte("serverhello "+strconv.FormatInt(start.Nanoseconds(), 10)+" "+strconv.FormatInt(time.Since(baseTime).Nanoseconds(), 10)))
		if err != nil {
			return err
		}
	default:
		err = c.Write(ctx, websocket.MessageText, []byte("error unknown opcode"))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s liveServer) writePump(ctx context.Context, c *websocket.Conn) error {
	clients.mu.Lock()
	clients.c = append(clients.c, func(msg []byte) {
		err := c.Write(ctx, websocket.MessageText, msg)
		if err != nil {
			log.Printf("failed to write to client: %v", err)
		}
	})
	clients.mu.Unlock()

	select {}
}