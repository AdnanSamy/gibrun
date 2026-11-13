const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const source = fs.readFileSync(__dirname + '/tabs.js', 'utf8');
// Exercise actual session functions; DOM behavior lives in tabs_test.cjs.
const prefix = source.slice(0, source.indexOf('  // Bootstrap owns'));
const hooks = {};
vm.runInNewContext(prefix + '\n hooks.readSession = readSession; hooks.persist = persist; hooks.restore = restore; })();', { hooks, document: { dispatchEvent() {}, querySelectorAll: () => [], createElement: () => ({ dataset: {}, setAttribute() {}, append() {} }), getElementById: () => ({ parentElement: { getBoundingClientRect: () => ({ left: 0, right: 0 }) } }) }, CustomEvent: class {} });
const current = 'gibrun.request-tabs.v1', legacy = 'solarclient.request-tabs.v1';
const saved = JSON.stringify({ version: 1, ids: ['b', 'a', 'b'], active: 'b', secret: 'DO NOT STORE' });
function storage(values) {
  return { getItem: key => values[key] ?? null, setItem: (key, value) => { values[key] = value; } };
}
const values = { [legacy]: saved };
const session = hooks.readSession(storage(values));
assert.deepEqual(JSON.parse(JSON.stringify(session)), { ids: ['b', 'a'], active: 'b' });
const tab = id => ({ panel: { querySelector: () => ({ elements: { id: { value: id } } }) } });
hooks.persist({ tabs: [tab('b'), tab('a')], active: tab('b'), restoring: false }, storage(values));
assert.deepEqual(JSON.parse(values[current]), { version: 1, ids: ['b', 'a'], active: 'b' });
assert.equal(values[legacy], saved);
assert.ok(!values[current].includes('DO NOT STORE'));
const newer = JSON.stringify({ version: 1, ids: ['new'], active: 'new' });
assert.equal(hooks.readSession(storage({ [current]: newer, [legacy]: saved })).active, 'new');
for (const value of ['', '{', 'null', JSON.stringify({ version: 1, ids: ['https://evil.test'], active: null }), JSON.stringify({ version: 1, ids: Array(101).fill('a'), active: null }), JSON.stringify({ version: 1, ids: ['a'], active: 'missing' })]) {
  assert.equal(hooks.readSession(storage({ [current]: value, [legacy]: saved })), null);
  assert.equal(hooks.readSession(storage({ [legacy]: value })), null);
}
assert.equal(hooks.readSession({ getItem() { throw Error('disabled'); } }), null);
hooks.persist({ tabs: [], active: tab(''), restoring: false }, { setItem() { throw Error('full'); } });
(async () => {
  const panel = id => ({ querySelector: () => ({ elements: { id: { value: id }, name: { value: id } } }), querySelectorAll: () => [], setAttribute() {}, replaceWith() {} });
  const initial = { panel: panel('a'), key: 1 };
  const state = { tabs: [initial], active: initial, restoring: true, revision: 0, serial: 1 };
  const calls = [];
  const view = { list: { replaceChildren() {}, append() {}, getBoundingClientRect: () => ({ left: 0, right: 100 }), scrollWidth: 100, clientWidth: 100, scrollLeft: 0 }, previous: {}, next: {} };
  const services = { storage: storage({}), htmx: { process() {} }, fetch: async (url, options) => {
    calls.push(url); assert.equal(options.credentials, 'same-origin'); assert.equal(options.redirect, 'error');
    return { ok: !url.endsWith('deleted'), text: async () => url.split('=')[1] };
  }, parser: { parseFromString: id => ({ querySelector: () => panel(id) }) } };
  await hooks.restore(state, view, services, { ids: ['b', 'a', 'deleted'], active: 'b' });
  assert.deepEqual(calls, ['/?id=b', '/?id=deleted']);
  assert.equal(state.active.panel.querySelector().elements.id.value, 'b');
  assert.equal(state.tabs.length, 2);
  assert.equal(state.restoring, false);
  console.log('tab session migration, precedence, validation, deduplication, restore and secret exclusion passed');
})().catch(error => { console.error(error); process.exitCode = 1; });
