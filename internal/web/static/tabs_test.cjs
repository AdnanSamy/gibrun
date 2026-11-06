const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const listeners = {};
class Node {
  constructor() { this.children = []; this.attributes = {}; }
  append(...nodes) { this.children.push(...nodes); }
  setAttribute(key, value) { this.attributes[key] = value; }
  replaceChildren() { this.children = []; }
  before() {}
  addEventListener() {}
  getBoundingClientRect() { return { left: 0, right: 190 }; }
  get parentElement() { return this; }
  focus() {}
  querySelectorAll() { return []; }
}
const panel = new Node();
const form = { elements: { id: { value: 'saved-1' }, name: { value: 'Example' } } };
panel.querySelector = () => form;
const nodes = [];
const hooks = {};
let warnings = 0;
vm.runInNewContext(fs.readFileSync(__dirname + '/tabs.js', 'utf8').replace("  register(document.querySelector('#main'));", "  hooks.refresh = refresh; hooks.tabs = tabs; register(document.querySelector('#main'));"), {
  hooks,
  ResizeObserver: class { observe() {} disconnect() {} },
  document: {
    createElement: () => { const node = new Node(); nodes.push(node); return node; },
    querySelector: () => panel,
    querySelectorAll: () => [],
    getElementById: () => new Node(),
    addEventListener: (name, callback) => { listeners[name] = callback; },
  },
  window: { addEventListener: (name, callback) => { listeners[name] = callback; } },
  htmx: { process() {} },
  alert: () => warnings++,
});
assert.equal(nodes[1].attributes.role, 'tablist');
assert.equal(nodes.find(node => node.attributes.role === 'tab').attributes['aria-selected'], 'true');
assert.equal(panel.attributes.role, 'tabpanel');
let prevented = false;
const unload = () => listeners.beforeunload({ preventDefault() { prevented = true; } });
unload(); assert.equal(prevented, false);
listeners.input({ target: { closest: () => form, name: 'body' } });
unload(); assert.equal(prevented, true);
listeners['htmx:beforeRequest']({ preventDefault() { throw Error('first request blocked'); } });
let blocked = false;
listeners['htmx:beforeRequest']({ preventDefault() { blocked = true; } });
assert.equal(blocked, true); assert.equal(warnings, 1);
listeners['htmx:afterRequest']();
function option(value, textContent) { return { value, textContent, cloneNode() { return option(this.value, this.textContent); } }; }
function select(value, options) { return { value, options, replaceChildren(...items) { this.options = items; }, append(item) { this.options.push(item); }, setCustomValidity(message) { this.validationMessage = message; } }; }
const states = [];
function editorPanel(id, collection, environment) {
  const editor = { elements: { id: { value: id }, name: { value: 'Unsaved name' }, collection: select(collection, []), environment: select(environment, []), body: { value: 'Unsaved body' } } };
  const state = {}, save = {}, files = { files: ['selected-file'] };
  const panel = { querySelector: selector => selector.startsWith('form') ? editor : selector === '#save-state' ? state : selector === '#save-request' ? save : null, querySelectorAll: () => [files] };
  states.push({ editor, state, save, files });
  return { panel, dirty: false };
}
hooks.tabs.splice(0, hooks.tabs.length, editorPanel('saved-1', 'c1', 'e1'), editorPanel('deleted', 'removed', 'gone'));
const fresh = { elements: { collection: select('', [option('', 'Unfiled'), option('c1', 'Renamed collection'), option('c2', 'New collection')]), environment: select('', [option('', 'No environment'), option('e1', 'Renamed environment')]) } };
const page = { querySelector: () => ({ querySelector: () => fresh }), querySelectorAll: () => [{ dataset: { id: 'saved-1' } }] };
hooks.refresh(page);
assert.equal(states[0].editor.elements.collection.value, 'c1');
assert.equal(states[0].editor.elements.collection.options[1].textContent, 'Renamed collection');
assert.equal(states[0].editor.elements.environment.options[1].textContent, 'Renamed environment');
assert.equal(states[1].editor.elements.collection.value, 'removed');
assert.match(states[1].editor.elements.collection.validationMessage, /Choose/);
assert.match(states[1].editor.elements.environment.validationMessage, /Choose/);
assert.equal(states[1].editor.elements.id.value, '');
assert.equal(states[1].save.textContent, 'Save as new request');
assert.equal(hooks.tabs[1].dirty, true);
for (const state of states) {
  assert.equal(state.editor.elements.body.value, 'Unsaved body');
  assert.equal(state.editor.elements.name.value, 'Unsaved name');
  assert.deepEqual(state.files.files, ['selected-file']);
}
states[1].editor.elements.collection.value = '';
hooks.refresh(page);
assert.equal(states[1].editor.elements.collection.validationMessage, '');
console.log('request tab semantics, guards, option reconciliation and deleted drafts passed');
