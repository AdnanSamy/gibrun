const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const listeners = {};
const cancel = { disabled: true };
const status = { textContent: '' };
const file = { files: ['selected-file'] };
const form = { matches: selector => selector === 'form[action="/request"]', draft: 'unsaved', file };
const xhr = {};
let aborts = 0;
const emit = (name, detail) => listeners[name]({ detail });
vm.runInNewContext(fs.readFileSync(__dirname + '/request.js', 'utf8'), {
  document: {
    addEventListener: (name, listener) => { listeners[name] = listener; },
    getElementById: id => id === 'cancel-request' ? cancel : status,
  },
  htmx: { trigger: (elt, name) => {
    assert.equal(elt, form);
    assert.equal(name, 'htmx:abort');
    aborts++;
    emit('htmx:afterRequest', { xhr, successful: false });
  } },
});
emit('htmx:beforeRequest', { elt: form, xhr, requestConfig: { parameters: { action: 'save' } } });
assert.equal(cancel.disabled, true);
emit('htmx:beforeRequest', { elt: form, xhr, requestConfig: { parameters: { action: 'send' } } });
assert.equal(cancel.disabled, false);
assert.equal(status.textContent, 'Sending…');
listeners.click({ target: { closest: () => cancel } });
assert.equal(aborts, 1);
assert.equal(cancel.disabled, true);
assert.match(status.textContent, /Cancelled.*side effects.*retained/);
assert.equal(form.draft, 'unsaved');
assert.equal(form.file, file);
assert.deepEqual(file.files, ['selected-file']);
const swap = { xhr, shouldSwap: true };
emit('htmx:beforeSwap', swap);
assert.equal(swap.shouldSwap, false);
const next = {};
emit('htmx:beforeRequest', { elt: form, xhr: next, requestConfig: { parameters: { action: 'send' } } });
emit('htmx:afterRequest', { xhr: next, successful: false });
assert.equal(cancel.disabled, true);
assert.match(status.textContent, /Request failed/);
const html = fs.readFileSync(__dirname + '/../page.html', 'utf8');
assert.match(html, /hx-disabled-elt="#send-request, #save-request"/);
assert.match(html, /src="\/static\/request.js"/);
console.log('request cancellation checks passed');
