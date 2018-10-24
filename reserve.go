package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"text/template"

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
		},
		'js': f => {
		return () => {
			return fetch(f, { cache: 'reload' })
				.then(r => r.blob())
				.then(blob => Promise.all([
					import(` + "`" + `${f}?live_module` + "`" + `),
					import(URL.createObjectURL(blob)),
				]))
				.then(mods => {
					const oldm = mods[0];
					const newm = mods[1];
					if (!oldm.__reserve_setters)
						location.reload(true);
					for (const k in newm) {
						const oldproto = oldm[k].prototype;
						const newproto = newm[k].prototype;
						if (oldproto) {
							for (const protok of Object.getOwnPropertyNames(oldproto)) {
								Object.defineProperty(oldproto, protok, { value: function (...args) {
									Object.setPrototypeOf(this, newproto);
									if (this.adopt)
										this.adopt(oldproto);
									this[protok](...args);
								} });
							}
						}
						const setter = oldm.__reserve_setters[k];
						if (!setter)
							location.reload(true);
						setter(newm[k]);
					}
				})
			};
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
			if (hooks[target]() !== false)
				return;
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

func jsWrapper(orig_filename string) string {
	f := template.JSEscapeString(orig_filename)
	return `
export * from "` + f + `"

import * as mod from "` + f + `"
let _default = mod.default
export {_default as default}

export const __reserve_setters = {}
for (const k in mod)
  __reserve_setters[k] = eval(` + "`" + `v => ${k == 'default' ? '_default' : k} = v` + "`" + `)
	`
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
		} else if _, exists := r.URL.Query()["live_module"]; exists {
			w.Header().Set("Content-Type", "application/javascript")
			w.Write([]byte(jsWrapper(r.URL.Path)))
		} else {
			fileServer.ServeHTTP(w, r)
			// w.Write([]byte("outer fn was here"))
		}
	})))
}
