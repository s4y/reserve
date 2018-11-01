package sse

import (
	"fmt"
	"net/http"
	"strings"
)

type event []byte

type Client chan event

type SSEServer struct {
	Id uint32

	clients []Client
	tasks   chan func()
}

func (b *SSEServer) subscribe(ch Client) {
	b.tasks <- func() {
		b.clients = append(b.clients, ch)
	}
}

func (b *SSEServer) unsubscribe(ch Client) {
	b.tasks <- func() {
		for i, cur_ch := range b.clients {
			if cur_ch == ch {
				b.clients = append(b.clients[:i], b.clients[i+1:]...)
				break
			}
		}
	}
}

func (s *SSEServer) Start() {
	s.tasks = make(chan func())
	go func() {
		for task := range s.tasks {
			task()
		}
	}()
}

func (s *SSEServer) Emit(targets []Client, event, data string) {
	s.tasks <- func() {
		m := "id:" + fmt.Sprint(s.Id) + "\nevent:" + event + "\n"
		for _, line := range strings.Split(data, "\n") {
			m += "data:" + line + "\n"
		}
		m += "\n"
		event := []byte(m)
		if targets == nil {
			targets = s.clients
		}
		for _, client := range targets {
			client <- event
		}
		s.Id++
	}
}

func (s *SSEServer) Broadcast(event, data string) {
	s.Emit(nil, event, data)
}

func (s *SSEServer) ServeHTTPCB(w http.ResponseWriter, r *http.Request, cb func(Client)) {
	flusher, _ := w.(http.Flusher)
	ch := make(Client)
	s.subscribe(ch)
	defer s.unsubscribe(ch)
	defer close(ch)
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.WriteHeader(200)
	if flusher != nil {
		flusher.Flush()
	}
	if cb != nil {
		go func() {
			cb(ch)
		}()
	}
	for {
		select {
		case event := <-ch:
			w.Write(event)
			if flusher != nil {
				flusher.Flush()
			}
		case <-w.(http.CloseNotifier).CloseNotify():
			return
		}
	}
}

func (s *SSEServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.ServeHTTPCB(w, r, nil)
}
