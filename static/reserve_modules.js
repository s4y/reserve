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

window.__reserve_hooks_by_extension.js = f => {
  let last_f = f;
  return f_new => {
    if (!window.__reserve_hot_modules || !window.__reserve_hot_modules[f])
      return false;
    const next_f = `${f_new}&raw`;
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
        if (oldm.default.__on_module_reloaded)
          newm.default.__on_module_reloaded = oldm.default.__on_module_reloaded;
        if (oldm.default.__file)
          newm.default.__file = oldm.default.__file;
        if (!Object.hasOwnProperty(oldm.default.prototype, 'adopt'))
          oldm.default.prototype.adopt = function(){};
        if (!Object.hasOwnProperty(newm.default.prototype, 'adopt'))
          newm.default.prototype.adopt = function(){};
        for (const k in newm) {
          const oldproto = oldm[k].prototype;
          const newproto = newm[k].prototype;
          if (oldproto) {
            for (const protok of Object.getOwnPropertyNames(oldproto)) {
              if (protok === 'constructor')
                continue;
              Object.defineProperty(oldproto, protok, { value: function (...args) {
                if (Object.getPrototypeOf(this) != oldproto)
                  return false;
                Object.setPrototypeOf(this, newproto);
                if (this.adopt && protok != 'adopt')
                  this.adopt(oldproto);
                return this[protok](...args);
              } });
            }
          }
          const setter = origm.__reserve_setters[k];
          if (!setter)
            location.reload(true);
          setter(newm[k]);

          if (newm.default.__on_module_reloaded) {
            for (const f of newm.default.__on_module_reloaded)
              f();
          }
        }
        return true;
      });
  };
};
