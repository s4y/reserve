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

package watcher

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rjeczalik/notify"
)

func hasHiddenComponent(p string) bool {
	rest := p
	for {
		if strings.HasPrefix(path.Base(rest), ".") {
			return true
		}
		next := path.Dir(rest)
		if next == rest {
			break
		}
		rest = next
	}
	return false
}

type Watcher struct {
	Changes chan string

	events chan notify.EventInfo
}

func NewWatcher(dir string) *Watcher {
	w := Watcher{}
	w.Changes = make(chan string)
	w.events = make(chan notify.EventInfo, 100)

	go func() {
		// A text editor may save a file in several steps, like creating a
		// temporary file and then renaming it on top of the original, or
		// creating a backup file and then deleting it.
		//
		// To squash the noise, the top-level loop waits for one event, then
		// sets a timeout and collects any additional events that arrive before
		// it fires, then dispatches events only for files which still exist at
		// the end. (As a result, deletes aren't reported; that's fine for the
		// use case of "reload files that change".)
		for {
			event := <-w.events
			touched := make(map[string]bool)
			handle := func(event notify.EventInfo) {
				touched[event.Path()] = true
			}
			handle(event)

			// 3ms felt right, but might not be.
			for timeout := time.After(3 * time.Millisecond); timeout != nil; {
				select {
				case event := <-w.events:
					handle(event)
				case <-timeout:
					timeout = nil
				}
			}

			absDir, _ := filepath.EvalSymlinks(dir)

			for path, _ := range touched {
				if hasHiddenComponent(path) ||
					// Vim backup files. This check can be tightened up if it's an
					// issue for anyone.
					strings.HasSuffix(path, "~") {
					continue
				}
				if _, err := os.Stat(path); err != nil {
					continue
				}
				relpath, err := filepath.Rel(absDir, path)
				if err != nil {
					log.Fatal(err)
				}
				w.Changes <- relpath
			}
		}
	}()

	err := notify.Watch(filepath.Join(dir, "..."), w.events, notify.All)
	if err != nil {
		log.Fatal(err)
	}
	return &w
}
