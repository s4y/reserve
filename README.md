# Reserve: a small, useful web server

Reserve is a web server which instantly reloads web pages (or parts of them) as you edit them. It also adds useful JavaScript APIs to, for instance, broadcast messages to other browsers.

Reserve is primarily intended for rapid prototyping of ideas, and for interactive multi-device installations — it doesn’t have built-in support for sessions or authentication, for instance.

## Getting started

If you’re on macOS, you can install reserve via Homebrew:

```shell
brew install s4y/reserve/reserve
```

Or, on any OS, ensure you have `go` and that your `GOPATH` is in your path (i.e. there's a line like `PATH="$PATH:$HOME/go/bin"` in your `.bashrc` or similar), then install reserve with `go install`:

```shell
go install github.com/s4y/reserve/reserve@latest
```

Then, run `reserve` in the directory which contains your project. It will print a link and then keep running (type Ctrl-C to exit):

```shell
> ls
index.html style.css
> reserve
http://127.0.0.1:8080/
```

The whole page will reload if almost any file changes on disk. But some kinds of files get special treatment. If a page includes CSS, like this:

```html
<!DOCTYPE html>
<link rel=stylesheet href=style.css>
```
…then, if `style.css` changes, the style will update without reloading the page.

## JavaScript API

### `reserve.broadcast(anything)`

This is one of reserve's the most useful features. You can broadcast any JSON-serializable message like this:

```javascript
reserve.broadcast({ foo: "bar" });
```

…and you can receive broadcasts like this:

```javascript
window.addEventListener("broadcast", e => {
  console.log(e.detail);
});
```

This lets you quickly build, say, a video player with a remote that you can open on your phone, or a little chat app, using only client-side JavaScript.

### `sourcechange` event

Reserve emits an event on `window` when a file changes on disk. You can call `.preventDefault()` on the event to stop reserve from reloading the whole page. For example:

```javascript

const load = () => fetch("whatever.json", {
  cache: "reload",
}).then(async r => console.log(await r.json()));

load();

window.addEventListener("sourcechange", e => {
  const changedPath = new URL(e.detail, location).pathname;
  if (changedPath == "/whatever.json") {
    e.preventDefault();
    load();
  }
})

```

## Options

By default, other computers cannot access the web server. You may specify an alternate port or IP address with the `-http` flag, or leave the IP address out to listen on all interfaces:

| To… | Run… |
| --- | ---- |
| …choose a different port | `reserve -http=127.0.0.1:8888` |
| …let other computers connect | `reserve -http=:8080` |

Letting other computers on the network connect can be great for prototyping with a friend (who can load the page on their own computer and watch it update), for testing on mobile devices, or for multi-screen experiences.

## Tips and Tricks

If you include a transition in your CSS, like this:

**style.css**:

```css
* { transition: all 0.2s; }
```

…then style changes will ✨animate✨.

## Advanced

Reserve includes **experimental** support for reloading JavaScript modules. Only modules with a single default export are supported.

For example, if you have the following files in your project:

**index.html**:

```html
<!DOCTYPE html>
<div id=count></div>
<script type=module>
import Counter from '/Counter.js'

let countElement = document.getElementById('count');
let counter = new Counter();

setInterval(() => {
  countElement.textContent = counter.nextNumber();
}, 1000);
</script>
```

**Counter.js**:

```javascript
// reserve:hot_reload

export default class Counter {
  constructor() { this.count = 0; }
  nextNumber() {
    return this.count++;
  }
}
```

Then, if you change `nextNumber()` to look like this:

```javascript
  nextNumber() {
    return this.count += 2;
  }
```

…the page immediately starts counting up by two without reloading or losing the count. (Note: Reserve will only attempt to reload a module if its first line is `// reserve:hot_reload`. Otherwise, it sticks to reloading the whole page.)

To reload a module, Reserve modifies the old class so that if any method is called on an object of that class, the object's prototype switches to the new version before the method runs. If the (new) class has an `adopt()` method, then `adopt()` runs just before the original method. `adopt()` can perform any work (e.g. recreating an element) to update the object to the new version.

## Authors

Reserve was created by [Sidney San Martín](https://s4y.us) but is open to contribution by others.
