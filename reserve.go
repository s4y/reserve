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

package reserve

import (
	"bufio"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"github.com/gorilla/websocket"
	"github.com/s4y/reserve/httpsuffixer"
	"github.com/s4y/reserve/static"
	"github.com/s4y/reserve/watcher"
)

type HTMLSuffixer struct {
	Suffix         []byte
	sentScriptTags bool
	buf            []byte
}

var doctypeMatcher = regexp.MustCompile(`^\s*<![^>\n]+>`)

func (t *HTMLSuffixer) Tweak(data []byte) []byte {
	if data != nil && !t.sentScriptTags {
		t.buf = append(t.buf, data...)
		if doctype := doctypeMatcher.Find(t.buf); doctype != nil {
			t.sentScriptTags = true
			return append(doctype, append(t.Suffix, t.buf[len(doctype):]...)...)
		}
		return nil
	}
	if !t.sentScriptTags {
		data = append(t.Suffix, append(t.buf, data...)...)
		t.sentScriptTags = true
		t.buf = nil
	}
	return data
}

var gStaticFiles = map[string][]byte{
	"/.reserve/reserve.js":         []byte(static.ReserveJs),
	"/.reserve/reserve_modules.js": []byte(static.ReserveModulesJs),
}

func jsWrapper(orig_filename string) string {
	f := template.JSEscapeString(orig_filename)
	return `
import * as mod from "` + f + `?raw"
let _default = mod.default
export {_default as default}

export const __reserve_setters = {
	default: v => _default = v,
}

if (typeof _default  === "function")
	_default.__on_module_reloaded = [];

if (!window.__reserve_hot_modules)
  window.__reserve_hot_modules = {};
window.__reserve_hot_modules[new URL("` + f + `", location.href).href] = true;
`
}

func isHotModule(path string) bool {
	if !strings.HasSuffix(path, ".js") {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	firstLine, _ := bufio.NewReader(file).ReadString('\n')
	return firstLine == "// reserve:hot_reload\n"
}

type ClientConnections struct {
	connections []*websocket.Conn
	lock        sync.Mutex
}

func (s *ClientConnections) add(c *websocket.Conn) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.connections = append(s.connections, c)
}

func (s *ClientConnections) remove(c *websocket.Conn) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for i, cur_conn := range s.connections {
		if cur_conn == c {
			s.connections = append(s.connections[:i], s.connections[i+1:]...)
			break
		}
	}
}

func (s *ClientConnections) broadcast(message interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, conn := range s.connections {
		conn.WriteJSON(message)
	}
}

type MessageToClient struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

func CreateServer(directory string) http.Handler {
	upgrader := websocket.Upgrader{}
	conns := ClientConnections{}

	suffixer := httpsuffixer.SuffixServer{func(content_type string) httpsuffixer.Tweaker {
		switch content_type {
		case "text/html":
			// Slice to remove trailing newline
			return &HTMLSuffixer{Suffix: []byte(static.FilterHtml[:len(static.FilterHtml)-1])}
		default:
			return nil
		}
	}}

	watcher := watcher.NewWatcher(directory)
	go func() {
		for change := range watcher.Changes {
			conns.broadcast(MessageToClient{
				Name:  "change",
				Value: change,
			})
		}
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			conns.broadcast(MessageToClient{
				Name:  "stdin",
				Value: scanner.Text(),
			})
		}
		os.Exit(0)
	}()

	fileServer := suffixer.WrapServer(http.FileServer(http.Dir(directory)))
	fileServer = func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			handler.ServeHTTP(w, r)
		})
	}(fileServer)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.reserve/ws" {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			conns.add(conn)
			for {
				var msg interface{}
				if err := conn.ReadJSON(&msg); err != nil {
					break
				}
				// TODO: Do something with messages from clients. For now, this loop
				// serves as a close waiter.
			}
			conns.remove(conn)
			conn.Close()
			return
		} else if _, exists := r.URL.Query()["raw"]; !exists && isHotModule(path.Join(".", r.URL.Path)) {
			w.Header().Set("Content-Type", "application/javascript")
			w.Write([]byte(jsWrapper(r.URL.Path)))
		} else if staticContent, ok := gStaticFiles[r.URL.Path]; ok {
			http.ServeContent(w, r, r.URL.Path, static.ModTime, strings.NewReader(string(staticContent)))
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})
}
