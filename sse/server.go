// Copyright 2019 The Reserve Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
