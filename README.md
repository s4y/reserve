# Reserve, a page-updating web server
Reserve is a web server, intended for local development use, which instantly reloads web pages (or parts of them) as you edit them.

## Getting started
Run `reserve` in the directory which contains your website. It will print a link and then keep running (type Ctrl-C to exit):

```
> ls
index.html style.css
> reserve
http://127.0.0.1:8080/
```

By default, other computers cannot visit your site. You may specify an alternate port or IP address with the `-http` flag:

```
# Listen locally on a different port:
> reserve -http=127.0.0.1:8888
# Listen *publicly* on all of your computer's IP addresses:
> reserve -http=:8080
```

Reserve will currently reload the whole page if its HTML file changes. If the page includes external CSS, like this:

**index.html**:

```html
<!DOCTYPE html>
<link rel=stylesheet href=style.css>
```
…then if `style.css` changes, the page's CSS will update without reloading the page as a whole.

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

…the page should immediately start counting up by two, without reloading or restarting the count.

If your class has an `adopt()` method, then it will be called on each instance the first time any method is called on that instance, before the method runs. It may perform any work (e.g. recreating an element) to update it to the new version.

## Current status

This project is in its early stages and has only been used by its creator. Please file bugs and suggestions!

## Authors

Reserve was created by [Sidney San Martín](https://s4y.us) but is open to contribution by others.
