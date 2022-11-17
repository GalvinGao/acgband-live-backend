package main

import "sync"

type Clients struct {
	c []func([]byte)

	mu sync.RWMutex
}

var clients = Clients{}

func fanout(msg []byte) {
	clients.mu.RLock()
	defer clients.mu.RUnlock()

	for _, c := range clients.c {
		c(msg)
	}
}
