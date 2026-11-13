const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const source = fs.readFileSync(__dirname + '/tabs.js', 'utf8');
const handlers = {};
let offset = 0, disconnected = false;
const list = { clientWidth: 300, scrollWidth: 900,
  get scrollLeft() { return offset; }, set scrollLeft(value) { offset = Math.max(0, Math.min(600, value)); },
  getBoundingClientRect: () => ({ left: 100, right: 400 }),
  addEventListener: (name, fn) => { handlers[name] = fn; }
};
const previous = {}, next = {}, strip = {};
let item = { left: 450, right: 640 };
vm.runInNewContext(source.slice(source.indexOf('  function updateOverflow('), source.indexOf('  function render(')) + `
const view = { list, previous, next };
updateOverflow(view);
handlers.resize = () => { updateOverflow(view); revealActive(active, view); };
handlers.wheel = event => wheelTabs(event, view);
handlers.scroll = () => updateOverflow(view);
`, {
  handlers, list, previous, next, strip, active: { key: 1 },
  document: { getElementById: () => ({ parentElement: { getBoundingClientRect: () => item } }), addEventListener: (name, fn) => { handlers[name] = fn; }, removeEventListener() {} },
  ResizeObserver: class { constructor(fn) { handlers.resize = fn; } observe() {} disconnect() { disconnected = true; } }
});
assert.equal(previous.hidden, false);
assert.equal(previous.disabled, true);
handlers.resize();
assert.equal(offset, 240); // Nearest edge, not whole-document scroll.
item = { left: 80, right: 270 };
handlers.resize();
assert.equal(offset, 220);
let prevented = false;
const wheel = extra => handlers.wheel({ deltaX: 0, deltaY: 80, deltaMode: 0, preventDefault() { prevented = true; }, ...extra });
wheel({}); assert.equal(offset, 220); assert.equal(prevented, false);
wheel({ shiftKey: true }); assert.equal(offset, 300); assert.equal(prevented, true);
prevented = false;
wheel({ ctrlKey: true, deltaX: 100 }); assert.equal(offset, 300); assert.equal(prevented, false);
list.scrollLeft = 600;
wheel({ deltaX: 100, deltaY: 0 }); assert.equal(prevented, false); assert.equal(next.disabled, true);
list.scrollWidth = 300; handlers.scroll(); assert.equal(previous.hidden, true); assert.equal(next.hidden, true);

console.log('tab overflow, nearest reveal, wheel boundaries and resize passed');
