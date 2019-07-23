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
	"strings"
	"text/template"

	"github.com/s4y/reserve/httpsuffixer"
	"github.com/s4y/reserve/sse"
	"github.com/s4y/reserve/static"
	"github.com/s4y/reserve/watcher"
)

var gFilters = map[string][]byte{
	"text/html": []byte(static.FilterHtml),
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

func CreateServer(directory string) http.Handler {
	changeServer := sse.Server{}

	suffixer := httpsuffixer.SuffixServer{func(content_type string) []byte {
		if filter, ok := gFilters[content_type]; ok {
			return filter
		}
		return nil
	}}

	watcher := watcher.NewWatcher(directory)
	go func() {
		for change := range watcher.Changes {
			changeServer.Broadcast(sse.Event{Name: "change", Data: "/" + change})
		}
	}()

	stdinServer := sse.Server{}
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			stdinServer.Broadcast(sse.Event{Name: "line", Data: scanner.Text()})
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
		if r.URL.Path == "/.reserve/changes" {
			changeServer.ServeHTTP(w, r)
		} else if r.URL.Path == "/.reserve/stdin" {
			stdinServer.ServeHTTP(w, r)
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
