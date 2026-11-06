// Run: node internal/web/static/highlight_test.cjs
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
function node() { return { children: [], append(...children) { this.children.push(...children); }, replaceChildren(...children) { this.children = children; }, set innerHTML(_) { throw Error('unsafe HTML'); } }; }
const context = { document: { createDocumentFragment: node, createTextNode: text => ({ textContent: text }), createElement: node } };
const source = fs.readFileSync(__dirname + '/app.js', 'utf8');
vm.createContext(context);
vm.runInContext(source.slice(source.indexOf('function highlightJSON'), source.indexOf('\n(() => {', source.indexOf('function highlightJSON'))), context);
const target = node();
const text = '{"html":"</span><script>alert(1)</script>","x":1,"ok":true,"escape":"\\\""}';
context.highlightJSON(target, text);
function contents(n) { return n.textContent ?? n.children.map(contents).join(''); }
assert.equal(contents(target), text);
assert.ok(target.children[0].children.some(n => n.className === 'json-string'));
console.log('highlight escaping passed');
