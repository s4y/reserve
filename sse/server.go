package sse

import (
	"net/http"
	"sync"
)

type Server struct {
	connections []Connection
	lock        sync.Mutex
}

func (s *Server) subscribe(c Connection) {
	s.lock.Lock()
	s.connections = append(s.connections, c)
	s.lock.Unlock()
}

func (s *Server) unsubscribe(c Connection) {
	s.lock.Lock()
	for i, cur_conn := range s.connections {
		if cur_conn == c {
			s.connections = append(s.connections[:i], s.connections[i+1:]...)
			break
		}
	}
	s.lock.Unlock()
}

func (s *Server) Broadcast(event Event) {
	data := event.Marshal()
	s.lock.Lock()
	for _, c := range s.connections {
		c <- data
	}
	s.lock.Unlock()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := make(Connection)
	s.subscribe(c)
	defer s.unsubscribe(c)
	c.ServeHTTP(w, r)
}
