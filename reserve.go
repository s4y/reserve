package main

import (
	"bufio"
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
	const newHookForExtension = {
		'css': f => {
			const fHref = new URL(f, location.href).href;
			for (let el of document.querySelectorAll('link[rel=stylesheet]')) {
				if (new URL(el.href, location.href).href != fHref)
					continue;
				return () => {
					return fetch(f, { cache: 'reload' })
						.then(r => r.blob())
						.then(blob => {
							el.href = URL.createObjectURL(blob);
						});
				};
				break;
			}
		}
	};
	const hooks = Object.create(null);

	const es = new EventSource("/.reserve/changes");
	es.addEventListener('change', e => {
		const target = e.data;
		if (!(target in hooks)) {
			const ext = target.split('/').pop().split('.').pop();
			if (newHookForExtension[ext])
				hooks[target] = newHookForExtension[ext](target);
		}
		if (hooks[target]) {
			return hooks[target]();
		}
		location.reload(true);
	});

	let wasOpen = false;
	es.addEventListener('open', e => {
		if (wasOpen)
			location.reload(true);
		wasOpen = true;
	});

	let stdin = new EventSource("/.reserve/stdin");
	stdin.addEventListener("line", e => {
		const ev = new CustomEvent('stdin');
		ev.data = e.data;
		window.dispatchEvent(ev);
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
			sseServer.Broadcast("change", "/"+change)
		}
	}()

	stdinServer := sseserver.SSEServer{}
	stdinServer.Start()
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			stdinServer.Broadcast("line", scanner.Text())
		}
	}()

	fileServer := suffixer.WrapServer(http.FileServer(http.Dir(cwd)))

	log.Fatal(http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.reserve/changes" {
			sseServer.ServeHTTP(w, r)
		} else if r.URL.Path == "/.reserve/stdin" {
			stdinServer.ServeHTTP(w, r)
		} else {
			fileServer.ServeHTTP(w, r)
			// w.Write([]byte("outer fn was here"))
		}
	})))
}
