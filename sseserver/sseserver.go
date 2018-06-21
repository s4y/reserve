package sseserver

import (
	"fmt"
	"net/http"
	"strings"
)

type event []byte

type SSEServer struct {
	Id uint32

	clients []chan event
	tasks   chan func()
}

func (b *SSEServer) subscribe(ch chan event) {
	b.tasks <- func() {
		b.clients = append(b.clients, ch)
	}
}

func (b *SSEServer) unsubscribe(ch chan event) {
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

func (s *SSEServer) Broadcast(event, data string) {
	s.tasks <- func() {
		m := "id:" + fmt.Sprint(s.Id) + "\nevent:" + event + "\n"
		for _, line := range strings.Split(data, "\n") {
			m += "data:" + line + "\n"
		}
		m += "\n"
		event := []byte(m)
		for _, client := range s.clients {
			client <- event
		}
		s.Id++
	}
}

func (s *SSEServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, _ := w.(http.Flusher)
	ch := make(chan event)
	s.subscribe(ch)
	defer s.unsubscribe(ch)
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.WriteHeader(200)
	if flusher != nil {
		flusher.Flush()
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
