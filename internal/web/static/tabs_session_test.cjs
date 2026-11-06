const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const source = fs.readFileSync(__dirname + '/tabs.js', 'utf8');
// Exercise the actual persistence/restore functions without browser dependencies.
const prefix = source.slice(0, source.indexOf('  let active, busy'));
async function run(stored, dirty = false, disabled = false) {
  const calls = [], writes = [], hooks = {};
  const panel = id => ({ id, secret: 'DO NOT STORE', querySelectorAll: () => [] });
  const initial = panel('a');
  vm.runInNewContext(prefix + `
    let serial = 0, revision = 0, busy = false;
    const identity = panel => panel.id;
    const uploads = () => false;
    let active = { panel: initial, dirty: false };
    tabs.push(active);
    const select = tab => { active = tab; };
    const render = persist;
    const form = panel => panel;
    hooks.edit = () => { revision++; active.dirty = true; };
    hooks.done = restore();
    hooks.tabs = tabs;
    hooks.active = () => active;
  })();`, {
    hooks, initial, sessionStorage: { getItem() { if (disabled) throw Error(); return stored; }, setItem(k, v) { if (disabled) throw Error(); writes.push(v); } },
    fetch: async (url, options) => { calls.push({ url, options }); if (dirty) { initial.secret = 'edited'; hooks.edit(); } return { ok: true, text: async () => url.includes('deleted') ? '' : decodeURIComponent(url.split('=')[1]) }; },
    DOMParser: class { parseFromString(id) { return { querySelector: () => panel(id) }; } },
    htmx: { process() {} }, document: { dispatchEvent() {} }, CustomEvent: class {},
  });
  await hooks.done;
  return { calls, writes, hooks };
}
(async () => {
  const result = await run(JSON.stringify({ version: 1, ids: ['b', 'a', 'b', 'deleted'], active: 'b' }));
  assert.deepEqual(result.calls.map(c => c.url), ['/?id=b', '/?id=deleted']);
  for (const { options } of result.calls) { assert.equal(options.credentials, 'same-origin'); assert.equal(options.redirect, 'error'); }
  assert.equal(result.hooks.active().panel.id, 'b');
  assert.deepEqual(JSON.parse(result.writes.at(-1)), { version: 1, ids: ['b', 'a'], active: 'b' });
  assert.ok(!result.writes.join('').includes('DO NOT STORE'));
  for (const value of ['{', 'null', JSON.stringify({ version: 1, ids: ['https://evil.test'], active: null }), JSON.stringify({ version: 1, ids: Array(101).fill('a'), active: null }), JSON.stringify({ version: 1, ids: ['a'], active: 'missing' })]) {
    assert.equal((await run(value)).calls.length, 0);
  }
  const closed = await run(JSON.stringify({ version: 1, ids: ['b'], active: 'b' }));
  assert.deepEqual(JSON.parse(closed.writes.at(-1)).ids, ['b']); // Stale address-bar ID must not reopen a closed tab.
  const edited = await run(JSON.stringify({ version: 1, ids: ['b'], active: 'b' }), true);
  assert.equal(edited.hooks.active().panel.id, 'a');
  assert.equal(edited.hooks.active().panel.secret, 'edited');
  assert.equal(edited.hooks.active().dirty, true);
  assert.equal((await run('', false, true)).writes.length, 0);
  console.log('tab session validation, deduplication, restore, deleted IDs and secret exclusion passed');
})().catch(error => { console.error(error); process.exitCode = 1; });
