const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const source = fs.readFileSync(__dirname + '/app.js', 'utf8');
const context = vm.createContext({});
vm.runInContext(source.slice(0, source.indexOf('(() => {')), context);
const filter = context.searchVisibility;
const rows = () => [
  { name: 'Accounts', folder: true, depth: 0 },
  { name: 'Nested', folder: true, depth: 1 },
  { name: 'List', method: 'GET', url: 'example.test/users', depth: 2 },
  { name: 'Create', method: 'POST', depth: 1 },
  { name: 'Other', method: 'DELETE', depth: 0 },
];
function visible(query, name = 'API') { return Array.from(filter(name, rows(), query).visible); }
test('case-insensitive name, method, URL matches retain only ancestor context', () => {
  for (const term of ['lIsT', ' get ', 'EXAMPLE.test/users']) assert.deepEqual(visible(term), [true, true, true, false, false]);
});
test('matching collection/folder includes descendants, not siblings', () => {
  assert.deepEqual(visible('accounts'), [true, true, true, true, false]);
  assert.deepEqual(visible('nested'), [true, true, true, false, false]);
  assert.deepEqual(visible('api'), [true, true, true, true, true]);
});
test('clear, empty results, empty collections, unfiled requests', () => {
  assert.deepEqual(visible('  '), [true, true, true, true, true]);
  assert.equal(filter('API', rows(), 'missing').show, false);
  assert.equal(filter('Empty', [], 'empty').show, true);
  assert.equal(filter('Empty', [], 'missing').show, false);
  assert.deepEqual(Array.from(filter('Unfiled requests', [{ name: 'Ping', depth: 0 }], 'PING').visible), [true]);
});
test('unlisted auth/body fields never participate', () => {
  assert.equal(filter('API', [{ name: 'Ping', depth: 0, auth: 'secret', body: 'secret' }], 'secret').show, false);
});
