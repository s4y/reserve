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
	"github.com/s4y/reserve/watcher"
)

var gFilters = map[string][]byte{
	"text/html": []byte(`
<script>
'use strict';
(() => {
	const cacheBustQuery = () => ` + "`" + `?cache_bust=${+new Date}` + "`" + `;
	const hookForExtension = {
		'js': f => {
			let last_f = f;
			return () => {
				if (!window.__reserve_hot_modules || !window.__reserve_hot_modules[f])
					return false;
				const next_f = ` + "`" + `${f}${cacheBustQuery()}&raw` + "`" + `;
				return Promise.all([
						import(f),
						import(last_f),
						import(next_f),
					])
					.then(mods => {
						last_f = next_f;
						const [origm, oldm, newm] = mods;
						if (!origm.__reserve_setters)
							location.reload(true);
						for (const k in newm) {
							const oldproto = oldm[k].prototype;
							const newproto = newm[k].prototype;
							if (oldproto) {
								for (const protok of Object.getOwnPropertyNames(oldproto)) {
									Object.defineProperty(oldproto, protok, { value: function (...args) {
										if (Object.getPrototypeOf(this) != oldproto)
											return false;
										Object.setPrototypeOf(this, newproto);
										if (this.adopt && protok != 'adopt')
											this.adopt(oldproto);
										this[protok](...args);
									} });
								}
							}
							const setter = origm.__reserve_setters[k];
							if (!setter)
								location.reload(true);
							setter(newm[k]);
						}
					});
			};
		}
	};
	const genericHook = f => {
		for (let el of document.querySelectorAll('link')) {
			if (el.href != f)
				continue;
			return () => {
				el.href = f + cacheBustQuery();
			};
			break;
		}
	};
	const hooks = Object.create(null);

	const es = new EventSource("/.reserve/changes");
	es.addEventListener('change', e => {
		const target = new URL(e.data, location.href).href;
		if (!(target in hooks)) {
			const ext = target.split('/').pop().split('.').pop();
			if (hookForExtension[ext])
				hooks[target] = hookForExtension[ext](target);
			else
				hooks[target] = genericHook(target);
		}
		Promise.resolve()
			.then(() => hooks[target]())
			.then(r => (r === false) && Promise.reject())
			.catch(() => {
				location.reload(true);
			});
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
export * from "` + f + `?raw"

import * as mod from "` + f + `?raw"
let _default = mod.default
export {_default as default}

export const __reserve_setters = {}
for (const k in mod)
  __reserve_setters[k] = eval(` + "`" + `v => ${k == 'default' ? '_default' : k} = v` + "`" + `)

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

func hasHiddenComponent(p string) bool {
	rest := p
	for {
		if strings.HasPrefix(path.Base(rest), ".") {
			return true
		}
		rest = path.Dir(rest)
		if rest == "." {
			break
		}
	}
	return false
}

func CreateServer(directory string) http.Handler {
	changeServer := sse.Server{}

	suffixer := httpsuffixer.SuffixServer{gFilters}

	watcher := watcher.NewWatcher(directory)
	go func() {
		for change := range watcher.Changes {
			if hasHiddenComponent(change) {
				continue
			}
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
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})
}
