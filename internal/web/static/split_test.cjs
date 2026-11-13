const { test } = require('node:test');
const assert = require('node:assert/strict');
const { clampHeight } = require('./split.js');

test('split height leaves both panes visible', () => {
  assert.equal(clampHeight(-100, 600), 120);
  assert.equal(clampHeight(900, 600), 480);
  assert.equal(clampHeight(300, 600), 300);
  assert.equal(clampHeight(0, 100), 50);
});
