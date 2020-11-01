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

'use strict';

window.__reserve_hooks_by_extension = {
  html: f => new_f => {
    // The current page, minus any query string or hash.
    let curpage = new URL(location.pathname, location.href).href;
    let target = f.replace(/index\.html$/, '');
    if (curpage == target)
      location.reload();
    return true;
  },
};

(() => {
  const ignorePats = [];
  const shouldIgnore = path => {
    for (const pat of ignorePats) {
      if (pat[0] == '/' && path.startsWith(pat))
        return true;
    }
    return false;
  };
  const reloadIgnoreFile = () => {
    fetch('/.reserveignore')
      .then(r => r.text())
      .then(text => {
        ignorePats.length = 0;
        for (const pat of text.split('\n')) {
          if (pat)
            ignorePats.push(pat);
        }
      });
  };
  reloadIgnoreFile();

  window.addEventListener('sourcechange', e => {
    const changedPath = new URL(e.detail, location.href).pathname;
    if (changedPath == '/.reserveignore') {
      reloadIgnoreFile();
      e.preventDefault();
      return;
    } else if (shouldIgnore(changedPath)) {
      e.preventDefault();
    }
  });

  const defaultHook = f => new_f => {
    let handled = false;
    for (let el of document.querySelectorAll('link')) {
      if (el.rel == "x-reserve-ignore") {
        const re = new RegExp(el.dataset.expr);
        if (re.test(f))
          handled = true;
        continue;
      }
      if (el.href != f && el.dataset.ohref != f)
        continue;
      if (!el.dataset.ohref)
        el.dataset.ohref = el.href;
      el.href = new_f;
      handled = true;
    }
    return handled;
  };
  const hooks = {};
  const cacheBustQuery = () => `?cache_bust=${+new Date}`;

  let queuedBroadcasts = [];
  const queueBroadcast = message => queuedBroadcasts.push(message);
  let broadcast = queueBroadcast;
  window.addEventListener('sendbroadcast', e => broadcast(e));

  const handleMessage = {
    change: path => {
      const target = new URL(`/${path}`, location.href).href;
      const cacheBustedTarget = target + cacheBustQuery();

      if (!window.dispatchEvent(new CustomEvent('sourcechange', {
        detail: target,
        cancelable: true,
      })))
        return;

      if (!(target in hooks)) {
        const ext = target.split('/').pop().split('.').pop();
        const genHook = window.__reserve_hooks_by_extension[ext];
        hooks[target] = genHook ? genHook(target) : () => Promise.resolve();
      }
      Promise.resolve()
        .then(() => hooks[target](cacheBustedTarget))
        .then(handled => handled || defaultHook(target)(cacheBustedTarget))
        .then(handled => handled || location.reload(true))
        .then(() => {
          for (const element of document.querySelectorAll('[data-reserve-notify-file="'+target+'"]'))
            element.dispatchEvent(new CustomEvent('sourcechange'));
        });
    },
    stdin: line => {
      const ev = new CustomEvent('stdin');
      ev.data = line;
      window.dispatchEvent(ev);
    },
    broadcast: message => {
      window.dispatchEvent(new CustomEvent('broadcast', { detail: message }))
    },
  };

  let wasOpen = false;
  const connect = () => {
    const ws = new WebSocket(`${location.protocol == 'https:' ? 'wss' : 'ws'}://${location.host}/.reserve/ws`);
    ws.onopen = e => {
      // if (wasOpen)
      //   location.reload(true);
      wasOpen = true;

      broadcast = e => {
        ws.send(JSON.stringify(e.detail));
      };
      while (queuedBroadcasts.length)
        broadcast(queuedBroadcasts.shift());
      };
    ws.onmessage = e => {
      const { name, value } = JSON.parse(e.data);
      handleMessage[name](value);
    };
    ws.onclose = e => {
      setTimeout(connect, 1000);
      broadcast = queueBroadcast;
    };
  };
  connect();
})();
