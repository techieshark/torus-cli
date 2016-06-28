/* eslint-env mocha */

'use strict';

var assert = require('assert');

var Context = require('../../lib/cli/context');
var Option = require('../../lib/cli/option');

describe('Option', function () {
  describe('constructor', function () {
    it('sets required if required', function () {
      var o = new Option('-p, --pretty <name>', 'description');

      assert.strictEqual(o.required, true, 'required');
      assert.strictEqual(o.optional, false, 'optional');
      assert.strictEqual(o.hasParam, true, 'hasParam');
      assert.strictEqual(o.bool, false, 'bool');
      assert.strictEqual(o.description, 'description');

      assert.strictEqual(o.name(), 'pretty');
      assert.strictEqual(o.shortcut(), 'p');
    });

    it('sets optional if optioned', function () {
      var o = new Option('-p, --pretty [name]');

      assert.strictEqual(o.required, false);
      assert.strictEqual(o.optional, true);
      assert.strictEqual(o.hasParam, true);
      assert.strictEqual(o.bool, false);

      assert.strictEqual(o.name(), 'pretty');
      assert.strictEqual(o.shortcut(), 'p');
    });

    it('throws an error if flags is not a string', function () {
      assert.throws(function () {
        var o = new Option(false); // eslint-disable-line
      }, /flags must be a string/);
    });

    it('throws an error for bad flag input', function () {
      assert.throws(function () {
        var o = new Option('-x [name]'); // eslint-disable-line
      }, /Flags do not match the regex/);
    });

    it('handles boolean', function () {
      var o = new Option('-s, --save', 'our description');

      assert.strictEqual(o.hasParam, false, 'hasParam');
      assert.strictEqual(o.bool, true, 'bool');
      assert.strictEqual(o.defaultValue, false);
    });

    it('handles boolean with --no-*', function () {
      var o = new Option('-n, --no-save', 'no saving');

      assert.strictEqual(o.hasParam, false);
      assert.strictEqual(o.bool, true, 'bool');
      assert.strictEqual(o.defaultValue, true, 'defaultValue');
    });
  });

  describe('#evaluate', function () {
    var c;
    beforeEach(function () {
      c = new Context({});
    });

    it('sets the value', function () {
      var o = new Option('-n, --name <name>', 'set name');

      o.evaluate(c, { name: 'hi' });

      assert.strictEqual(o.value, 'hi');
      assert.strictEqual(c.option('name'), o);
    });

    it('sets the value with shortcut', function () {
      var o = new Option('-n, --name <name>', 'set name');

      o.evaluate(c, { n: 'hi' });
      assert.strictEqual(o.value, 'hi');
      assert.strictEqual(c.option('name'), o);
    });

    it('sets the default value', function () {
      var o = new Option('-n, --name [name]', 'set name', 'joe');

      o.evaluate(c, {});

      assert.strictEqual(o.value, 'joe');
      assert.strictEqual(c.option('name'), o);
    });

    it('sets as undefined if val is a bool and default is not', function () {
      var o = new Option('-n, --name [name]', 'set name', null);

      o.evaluate(c, { n: true });

      assert.strictEqual(o.value, null);
      assert.strictEqual(c.option('name'), o);
    });

    it('sets as undefined if not bool', function () {
      var o = new Option('-n, --name [name]', 'set name');

      o.evaluate(c, {});

      assert.strictEqual(o.value, undefined);
      assert.strictEqual(c.option('name'), o);
    });

    it('sets as false if bool', function () {
      var o = new Option('-n, --no', 'stuff');

      o.evaluate(c, {});

      assert.strictEqual(o.value, false);
      assert.strictEqual(c.option('no'), o);
    });

    it('sets as true if no-bool', function () {
      var o = new Option('-x, --no-x', 'stuff');

      o.evaluate(c, {});
      assert.strictEqual(o.value, true);
      assert.strictEqual(c.option('x'), o);
    });
  });
});