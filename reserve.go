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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

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

const href = new URL("` + f + `", location.href).href;

if (typeof _default  === "function") {
	_default.__on_module_reloaded = [];
	_default.__file = href;
}

if (!window.__reserve_hot_modules)
  window.__reserve_hot_modules = {};
window.__reserve_hot_modules[href] = true;
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

type ClientConnections struct {
	connections []chan func(*websocket.Conn)
	lock        sync.Mutex
}

func (s *ClientConnections) add(c chan func(*websocket.Conn)) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.connections = append(s.connections, c)
}

func (s *ClientConnections) remove(c chan func(*websocket.Conn)) {
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
		conn <- func(c *websocket.Conn) {
			c.WriteJSON(message)
		}
	}
}

type Message struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

type Server struct {
	Dir       http.Dir
	ReadStdin bool

	handler http.Handler
}

func wrapConnection(c *websocket.Conn) chan func(*websocket.Conn) {
	ch := make(chan func(*websocket.Conn), 16)
	go func() {
		for f := range ch {
			f(c)
		}
	}()
	return ch
}

type minLastModifiedResponseWriter struct {
	http.ResponseWriter
}

var startupTime = time.Time.Round(time.Now().UTC(), time.Second)

func (w *minLastModifiedResponseWriter) WriteHeader(statusCode int) {
	if lm, err := time.Parse(http.TimeFormat, w.Header().Get("Last-Modified")); err != nil || lm.Before(startupTime) {
		w.Header().Set("Last-Modified", startupTime.Format(http.TimeFormat))
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func ensureMinLastModifiedTime(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if clientTime, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since")); err == nil {
			if clientTime.Before(startupTime) {
				r.Header.Del("If-Modified-Since")
			}
		}
		next.ServeHTTP(&minLastModifiedResponseWriter{w}, r)
	})
}

func serveJSONDirectoryListing(path string, w http.ResponseWriter, r *http.Request) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	type fileInfo struct {
		Name string `json:"name"`
	}
	fileInfos := []fileInfo{}
	for _, f := range files {
		name := f.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		fileInfos = append(fileInfos, fileInfo{Name: name})
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(fileInfos)
}

func (s *Server) start() {
	upgrader := websocket.Upgrader{}
	conns := ClientConnections{}

	suffixer := httpsuffixer.SuffixServer{
		NewTweaker: func(content_type string) httpsuffixer.Tweaker {
			switch content_type {
			case "text/html":
				// Slice to remove trailing newline
				return &HTMLSuffixer{Suffix: []byte(static.FilterHtml[:len(static.FilterHtml)-1])}
			default:
				return nil
			}
		}}

	absPath, _ := filepath.Abs(string(s.Dir))
	watcher := watcher.NewWatcher(absPath)
	go func() {
		for change := range watcher.Changes {
			conns.broadcast(Message{
				Name:  "change",
				Value: change,
			})
		}
	}()

	if s.ReadStdin {
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				conns.broadcast(Message{
					Name:  "stdin",
					Value: scanner.Text(),
				})
			}
			os.Exit(0)
		}()
	}

	fileServer := ensureMinLastModifiedTime(http.FileServer(s.Dir))
	suffixServer := suffixer.WrapServer(fileServer)
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		if _, exists := r.URL.Query()["raw"]; exists {
			fileServer.ServeHTTP(w, r)
		} else {
			suffixServer.ServeHTTP(w, r)
		}
	})

	s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fsPath := path.Join(absPath, r.URL.Path)
		if r.URL.Path == "/.reserve/ws" {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			ch := wrapConnection(conn)
			defer close(ch)
			conns.add(ch)
			defer conns.remove(ch)
			for {
				var msg Message
				if err := conn.ReadJSON(&msg); err != nil {
					break
				}
				switch msg.Name {
				case "broadcast":
					conns.broadcast(msg)
				case "ping":
					startTime, _ := msg.Value.(float64)
					ch <- func(conn *websocket.Conn) {
						conn.WriteJSON(Message{"pong", struct {
							StartTime  float64 `json:"startTime"`
							ServerTime int64   `json:"serverTime"`
						}{startTime, time.Now().UnixNano() / int64(time.Millisecond)}})
					}
				default:
					continue
				}
			}
			return
		} else if _, exists := r.URL.Query()["raw"]; !exists && isHotModule(fsPath) {
			w.Header().Set("Content-Type", "application/javascript")
			w.Write([]byte(jsWrapper(r.URL.Path)))
		} else if staticContent, ok := gStaticFiles[r.URL.Path]; ok {
			http.ServeContent(w, r, r.URL.Path, static.ModTime, strings.NewReader(string(staticContent)))
		} else if r.URL.Path == "/.reserveignore" && !fileExists(fsPath) {
			// Suppress 404s in the browser console
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Write([]byte{})
		} else {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			if _, wantJSON := r.URL.Query()["json"]; wantJSON {
				stat, err := os.Stat(fsPath)
				if err != nil {
					fmt.Println(err)
				}
				if stat != nil && stat.IsDir() {
					err := serveJSONDirectoryListing(fsPath, w, r)
					if err == nil {
						return
					}
				}
			}
			server.ServeHTTP(w, r)
		}
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.handler == nil {
		s.start()
	}
	s.handler.ServeHTTP(w, r)
}

func FileServer(directory http.Dir) *Server {
	return &Server{
		Dir: directory,
	}
}
