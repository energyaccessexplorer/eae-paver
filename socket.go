package main

import (
	"context"
	"fmt"
	"github.com/coder/websocket"
	"net/http"
	"time"
)

var SOCKET_ACCEPT_PATTERN string

var socket_table = map[string]*websocket.Conn{}

func socket_write(s *websocket.Conn, m string, r *http.Request) string {
	if r == nil {
		logger.Println("socket_write: got a nil request.")
		return ""
	}

	s.Write(r.Context(), websocket.MessageText, []byte(m))

	return m
}

func socket_destroy(id string, s *websocket.Conn, m string) {
	s.Close(websocket.StatusNormalClosure, m)
	delete(socket_table, id)
}

func socket_create(id string, w http.ResponseWriter, r *http.Request) {
	s, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{SOCKET_ACCEPT_PATTERN},
	})

	if err != nil {
		logger.Println(err.Error())
		return
	}

	socket_table[id] = s

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
	defer cancel()

	select {
	case <-ctx.Done():
		socket_destroy(id, s, fmt.Sprintf("timed out - %v", ctx.Err()))
	}
}
