package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/rjeczalik/notify"
)

type handler func(http.ResponseWriter, *http.Request)

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h(w, r)
}

func main() {
	host := "127.0.0.1:8080"
	fmt.Printf("http://%s/\n", host)

	ln, err := net.Listen("tcp", host)
	if err != nil {
		log.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	clients := []chan string{}
	fschannel := make(chan notify.EventInfo, 100)

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
		for event := range fschannel {
			touched := make(map[string]bool)
			handle := func(event notify.EventInfo) {
				touched[event.Path()] = true
			}
			handle(event)

			// 3ms felt right, but might not be.
			for timeout := time.After(3 * time.Millisecond); timeout != nil; {
				select {
				case event := <-fschannel:
					handle(event)
				case <-timeout:
					timeout = nil
				}
			}

			for path, _ := range touched {
				if _, err := os.Stat(path); err != nil {
					continue
				}
				fmt.Println(path)
				for _, client := range clients {
					client <- path
				}
			}
		}
	}()

	ServeSSE := func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-w.(http.CloseNotifier).CloseNotify():
			return
		}
	}

	err = notify.Watch(cwd, fschannel, notify.All)
	if err != nil {
		log.Fatal(err)
	}

	fileServer := http.FileServer(http.Dir(cwd))

	log.Fatal(http.Serve(ln, handler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		if r.URL.Path == "/.reserve/changes" {
			ServeSSE(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})))
}
