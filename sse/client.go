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
	"fmt"
	"net/http"
	"strings"
)

type Event struct {
	Id   *int
	Name string
	Data string
}

func (e Event) Marshal() []byte {
	m := ""
	if e.Id != nil {
		m += "id:" + fmt.Sprint(*e.Id) + "\n"
	}
	m += "event:" + e.Name + "\n"
	for _, line := range strings.Split(e.Data, "\n") {
		m += "data:" + line + "\n"
	}
	m += "\n"
	return []byte(m)
}

type Connection chan []byte

func (c *Connection) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, _ := w.(http.Flusher)
	defer close(*c)
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.WriteHeader(200)
	if flusher != nil {
		flusher.Flush()
	}
	for {
		select {
		case event, ok := <-*c:
			if !ok {
				return
			}
			w.Write(event)
			if flusher != nil {
				flusher.Flush()
			}
		case <-w.(http.CloseNotifier).CloseNotify():
			return
		}
	}
}
