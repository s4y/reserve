package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"./httpsuffixer"
	"./sseserver"
	"./watcher"
)

var gFilters = map[string][]byte{
	"text/html": []byte(`
<script>
'use strict';
(() => {
	const es = new EventSource("/.reserve/changes");
	es.addEventListener('change', e => {
		location.reload(true);
	});

	let wasOpen = false;
	es.addEventListener('open', e => {
		if (wasOpen)
			location.reload(true);
		wasOpen = true;
	});
})();
</script>
`),
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

	sseServer := sseserver.SSEServer{}
	sseServer.Start()

	suffixer := httpsuffixer.SuffixServer{gFilters}

	watcher := watcher.NewWatcher(cwd)
	go func() {
		for change := range watcher.Changes {
			sseServer.Broadcast("change", change)
		}
	}()

	fileServer := suffixer.WrapServer(http.FileServer(http.Dir(cwd)))

	log.Fatal(http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		if r.URL.Path == "/.reserve/changes" {
			sseServer.ServeHTTP(w, r)
		} else {
			fileServer.ServeHTTP(w, r)
			// w.Write([]byte("outer fn was here"))
		}
	})))
}
