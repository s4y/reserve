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

// Code generated by generate_static.go; DO NOT EDIT.

package static

import "time"

var ModTime = time.Unix(0, 1591637516512038000)

const FilterHtml = "<script src=\"/.reserve/reserve.js\"></script><script type=module src=\"/.reserve/reserve_modules.js\"></script>\n"
const ReserveJs = "// Copyright 2019 The Reserve Authors\n//\n// Licensed under the Apache License, Version 2.0 (the \"License\");\n// you may not use this file except in compliance with the License.\n// You may obtain a copy of the License at\n//\n//     https://www.apache.org/licenses/LICENSE-2.0\n//\n// Unless required by applicable law or agreed to in writing, software\n// distributed under the License is distributed on an \"AS IS\" BASIS,\n// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\n// See the License for the specific language governing permissions and\n// limitations under the License.\n\n'use strict';\n\nwindow.__reserve_hooks_by_extension = {\n  html: f => new_f => {\n    // The current page, minus any query string or hash.\n    let curpage = new URL(location.pathname, location.href).href;\n    let target = f.replace(/index\\.html$/, '');\n    if (curpage == target)\n      location.reload(true);\n  },\n};\n\n(() => {\n  const defaultHook = f => new_f => {\n    let handled = false;\n    for (let el of document.querySelectorAll('link')) {\n      if (el.rel == \"x-reserve-ignore\") {\n        const re = new RegExp(el.dataset.expr);\n        if (re.test(f))\n          handled = true;\n        continue;\n      }\n      if (el.href != f && el.dataset.ohref != f)\n        continue;\n      if (!el.dataset.ohref)\n        el.dataset.ohref = el.href;\n      el.href = new_f;\n      handled = true;\n    }\n    return handled;\n  };\n  const hooks = {};\n  const cacheBustQuery = () => `?cache_bust=${+new Date}`;\n\n  let queuedBroadcasts = [];\n  const queueBroadcast = message => queuedBroadcasts.push(message);\n  let broadcast = queueBroadcast;\n  window.addEventListener('sendbroadcast', e => broadcast(e));\n\n  const handleMessage = {\n    change: path => {\n      const target = new URL(path, location.href).href;\n      const cacheBustedTarget = target + cacheBustQuery();\n\n      if (!(target in hooks)) {\n        const ext = target.split('/').pop().split('.').pop();\n        const genHook = window.__reserve_hooks_by_extension[ext];\n        hooks[target] = genHook ? genHook(target) : () => Promise.resolve();\n      }\n      Promise.resolve()\n        .then(() => hooks[target](cacheBustedTarget))\n        .then(handled => handled || defaultHook(target)(cacheBustedTarget))\n        .then(handled => handled || location.reload(true));\n    },\n    stdin: line => {\n      const ev = new CustomEvent('stdin');\n      ev.data = line;\n      window.dispatchEvent(ev);\n    },\n    broadcast: message => {\n      window.dispatchEvent(new CustomEvent('broadcast', { detail: message }))\n    },\n  };\n\n  let wasOpen = false;\n  const connect = () => {\n    const ws = new WebSocket(`ws://${location.host}/.reserve/ws`);\n    ws.onopen = e => {\n      if (wasOpen)\n        location.reload(true);\n      wasOpen = true;\n\n      broadcast = e => {\n        ws.send(JSON.stringify(e.detail));\n      };\n      while (queuedBroadcasts.length)\n        broadcast(queuedBroadcasts.shift());\n      };\n    ws.onmessage = e => {\n      const { name, value } = JSON.parse(e.data);\n      handleMessage[name](value);\n    };\n    ws.onclose = e => {\n      setTimeout(connect, 1000);\n      broadcast = queueBroadcast;\n    };\n  };\n  connect();\n})();\n"
const ReserveModulesJs = "// Copyright 2019 The Reserve Authors\n//\n// Licensed under the Apache License, Version 2.0 (the \"License\");\n// you may not use this file except in compliance with the License.\n// You may obtain a copy of the License at\n//\n//     https://www.apache.org/licenses/LICENSE-2.0\n//\n// Unless required by applicable law or agreed to in writing, software\n// distributed under the License is distributed on an \"AS IS\" BASIS,\n// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\n// See the License for the specific language governing permissions and\n// limitations under the License.\n\nwindow.__reserve_hooks_by_extension.js = f => {\n  let last_f = f;\n  return f_new => {\n    if (!window.__reserve_hot_modules || !window.__reserve_hot_modules[f])\n      return false;\n    const next_f = `${f_new}&raw`;\n    return Promise.all([\n        import(f),\n        import(last_f),\n        import(next_f),\n      ])\n      .then(mods => {\n        last_f = next_f;\n        const [origm, oldm, newm] = mods;\n        if (!origm.__reserve_setters)\n          location.reload(true);\n        if (oldm.default.__on_module_reloaded)\n          newm.default.__on_module_reloaded = oldm.default.__on_module_reloaded;\n        if (oldm.default.__file)\n          newm.default.__file = oldm.default.__file;\n        if (!Object.hasOwnProperty(oldm.default.prototype, 'adopt'))\n          oldm.default.prototype.adopt = function(){};\n        if (!Object.hasOwnProperty(newm.default.prototype, 'adopt'))\n          newm.default.prototype.adopt = function(){};\n        for (const k in newm) {\n          const oldproto = oldm[k].prototype;\n          const newproto = newm[k].prototype;\n          if (oldproto) {\n            for (const protok of Object.getOwnPropertyNames(oldproto)) {\n              if (protok === 'constructor')\n                continue;\n              Object.defineProperty(oldproto, protok, { value: function (...args) {\n                if (Object.getPrototypeOf(this) != oldproto)\n                  return false;\n                Object.setPrototypeOf(this, newproto);\n                if (this.adopt && protok != 'adopt')\n                  this.adopt(oldproto);\n                return this[protok](...args);\n              } });\n            }\n          }\n          const setter = origm.__reserve_setters[k];\n          if (!setter)\n            location.reload(true);\n          setter(newm[k]);\n\n          if (newm.default.__on_module_reloaded) {\n            for (const f of newm.default.__on_module_reloaded)\n              f();\n          }\n        }\n        return true;\n      });\n  };\n};\n"
