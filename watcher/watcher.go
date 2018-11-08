package watcher

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/rjeczalik/notify"
)

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

			for path, _ := range touched {
				if _, err := os.Stat(path); err != nil {
					continue
				}
				relpath, err := filepath.Rel(dir, path)
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
