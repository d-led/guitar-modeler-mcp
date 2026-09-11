#!/usr/bin/env node
// extract-ge100pro-catalog.cjs
//
// Generates internal/mooer/ge100pro_catalog_data.go from the model tables that
// "Mooer Studio For GE100 Pro" ships in its renderer bundle:
//
//   out/renderer/assets/cab-*.js      AMP_TYPE_LIST / CAB_TYPE and their
//                                     display names (AMP_NAME_EN, CAB_NAME_EN)
//   out/renderer/assets/index-*.js    each module's model list with its real
//                                     hardware reference (effect_*_list,
//                                     *_EFFECT_NAME_EN)
//   out/renderer/assets/ns-*.js       the noise gate module's three models
//
// The editor numbers a module's models by their position in these lists, which
// is exactly the index a preset stores in its slot records, so these tables are
// the ground truth for writing a preset the device will load the intended model
// from. They are factual data (model names and the hardware they emulate), not
// creative expression, and are reproduced as plain Go tables with attribution.
//
// Usage: node scripts/extract-ge100pro-catalog.cjs <app-or-asar> <out.go>

'use strict';

const fs = require('fs');
const path = require('path');

const input = process.argv[2];
const outPath = process.argv[3];
if (!input || !outPath) {
  console.error('usage: node extract-ge100pro-catalog.cjs <app-or-asar> <out.go>');
  process.exit(1);
}

// ---------------------------------------------------------------- asar input

function asarPath(target) {
  if (target.endsWith('.asar')) return target;
  return path.join(target, 'Contents', 'Resources', 'app.asar');
}

// asar is a pickle header, the JSON directory, then the file bytes. The
// directory is not NUL terminated, so the end of the JSON object - found by
// bracket matching - is also where the file bytes start.
function readAsar(file) {
  const data = fs.readFileSync(file);
  const directoryEnd = endOfObject(data, 16);
  const directory = JSON.parse(data.subarray(16, directoryEnd).toString('utf8'));
  const base = directoryEnd;

  const files = new Map();
  const walk = (node, prefix) => {
    for (const [name, entry] of Object.entries(node.files || {})) {
      const p = `${prefix}/${name}`;
      if (entry.files) walk(entry, p);
      else if (entry.offset !== undefined) files.set(p, data.subarray(base + Number(entry.offset), base + Number(entry.offset) + Number(entry.size)));
    }
  };
  walk(directory, '');
  return files;
}

// endOfObject returns the offset just past the JSON object starting at or after
// `from`.
function endOfObject(data, from) {
  let i = data.indexOf(0x7b /* { */, from);
  if (i < 0) throw new Error('no asar directory in the archive');
  let depth = 0;
  let inString = false;
  for (; i < data.length; i++) {
    const c = data[i];
    if (inString) {
      if (c === 0x5c /* \ */) { i++; continue; }
      if (c === 0x22) inString = false;
      continue;
    }
    if (c === 0x22) { inString = true; continue; }
    if (c === 0x7b) depth++;
    else if (c === 0x7d /* } */ && --depth === 0) return i + 1;
  }
  throw new Error('unterminated asar directory');
}

const files = readAsar(asarPath(input));

// The tables are spread over the renderer bundles (the amp and cab lists live in
// the cab editor's chunk, the module lists in the main one), so every renderer
// bundle is a source.
const sources = [...files.keys()]
  .filter((p) => p.startsWith('/out/renderer/assets/') && p.endsWith('.js'))
  .map((p) => files.get(p).toString('utf8'));
if (sources.length === 0) throw new Error('no renderer bundles in the archive');

// ------------------------------------------------------- JS literal reading

// The bundles are plain ES modules: the tables are object/array literals, with
// enum members referenced by name. Read them by locating the assignment and
// scanning to its matching bracket, then evaluating the literal with a scope
// that resolves the bundled enums.
function literalIn(text, name) {
  const m = new RegExp('const ' + name + '\\s*=\\s*', 'm').exec(text);
  if (!m) return undefined;
  let i = m.index + m[0].length;
  const start = i;
  if (!'{['.includes(text[i])) {
    while (i < text.length && !',;\n'.includes(text[i])) i++;
    return text.slice(start, i);
  }
  let depth = 0;
  let str = null;
  let started = false;
  for (; i < text.length; i++) {
    const c = text[i];
    if (str) {
      if (c === '\\') { i++; continue; }
      if (c === str) str = null;
      continue;
    }
    if (c === '"' || c === "'" || c === '`') { str = c; continue; }
    if (c === '/' && text[i + 1] === '/') { while (i < text.length && text[i] !== '\n') i++; continue; }
    if (c === '{' || c === '[' || c === '(') { depth++; started = true; continue; }
    if (c === '}' || c === ']' || c === ')') { depth--; if (started && depth === 0) { i++; break; } continue; }
  }
  return text.slice(start, i);
}

// Identifiers that live in other bundles resolve to undefined rather than
// failing the read.
const ANY = new Proxy(function () {}, { get: () => ANY, has: () => true, apply: () => ANY, construct: () => ANY });

const ENUMS = {};
for (const text of sources) {
  for (const m of text.matchAll(/([A-Za-z_$][\w$]*?)2\["([A-Za-z0-9_$]+)"\]\s*=\s*(\d+)/g)) {
    const [, base, member, value] = m;
    (ENUMS[base] = ENUMS[base] || {})[member] = Number(value);
  }
}
const scope = new Proxy(ENUMS, {
  has: () => true,
  get: (target, key) => {
    if (key === Symbol.unscopables) return undefined;
    return key in target ? target[key] : ANY;
  },
});

function table(name) {
  for (const text of sources) {
    const literal = literalIn(text, name);
    if (literal === undefined) continue;
    return new Function('scope', 'with(scope){return (' + literal + ')}')(scope);
  }
  throw new Error(`table ${name} not found in the bundles`);
}

// ------------------------------------------------------------- catalog build

// Parameter labels are i18n keys (G_DS_VOLUME, NS_SENS, ...); the editor's
// parameter name maps carry the short on-screen text.
const paramLabels = collectParamLabels();
function collectParamLabels() {
  const labels = {};
  for (const text of sources) {
    for (const m of text.matchAll(/const ([A-Z0-9_]*(?:PARAM_NAME_EN|PARAM_EN))\s*=\s*\{/g)) {
      Object.assign(labels, table(m[1]));
    }
  }
  return labels;
}

function paramLabel(label) {
  return paramLabels[label] || label;
}

const ampTypes = table('AMP_TYPE_LIST');
const ampNames = table('AMP_NAME_EN');
const cabTypes = table('CAB_TYPE');
const cabNames = table('CAB_NAME_EN');

const ampCount = Number(table('AMP_GNR_START')); // user capture slots start here
const cabCount = Number(table('CAB_GIR_START')); // user IR slots start here

const isPlaceholder = (name) => !name || /^EMPTY/.test(name);

function displayName(map, key) {
  return map[key] || key;
}

const amps = ampTypes
  .slice(0, ampCount)
  .map((t) => displayName(ampNames, t.name))
  .filter((n) => !isPlaceholder(n));

const cabs = cabTypes
  .slice(0, cabCount)
  .map((c) => displayName(cabNames, c.name))
  .filter((n) => !isPlaceholder(n));

// Each module's models in wire index order. The editor groups them by
// effect_group; a device module takes the group of the same name, and the model
// selector's hidden "Type" parameter is the index in this list.
const moduleGroups = [
  { module: 'fx', list: 'effect_fx_list', names: 'FX_EFFECT_NAME_EN' },
  { module: 'od', list: 'effect_ds_list', names: 'DS_EFFECT_NAME_EN' },
  { module: 'eq', list: 'effect_eq_list', names: 'EQ_EFFECT_NAME_EN' },
  { module: 'mod', list: 'effect_mod_list', names: 'MOD_EFFECT_NAME_EN' },
  { module: 'delay', list: 'effect_delay_list', names: 'DELAY_EFFECT_NAME_EN' },
  { module: 'reverb', list: 'effect_reverb_list', names: 'REVERB_EFFECT_NAME_EN' },
];

const effects = new Map();
for (const { module, list, names } of moduleGroups) {
  const entries = table(list);
  const nameMap = table(names);
  effects.set(
    module,
    entries
      .map((e) => ({
        name: displayName(nameMap, e.effect_name),
        reference: e.reference || '',
        params: (e.effect_param_list || []).filter((p) => !p.hidden).map((p) => paramLabel(p.label)),
      }))
      .filter((e) => !isPlaceholder(e.name)),
  );
}

// The noise gate module keeps its three models in the editor's own NS table.
const nsEntries = table('NS_EFFECT_LIST');
const nsNames = table('NS_EFFECT_NAME_EN');
effects.set(
  'ns',
  nsEntries
    .map((e) => ({
      name: displayName(nsNames, e.effect_name),
      reference: e.reference || '',
      params: (e.effect_param_list || []).filter((p) => !p.hidden).map((p) => paramLabel(p.label)),
    }))
    .filter((e) => !isPlaceholder(e.name)),
);

// ------------------------------------------------------------------- output

const lines = [];
lines.push('// Code generated by scripts/extract-ge100pro-catalog.cjs; DO NOT EDIT.');
lines.push('//');
lines.push('// Model tables for the Mooer GE100 Pro, read from the tables Mooer Studio');
lines.push('// For GE100 Pro ships in its own renderer bundle. The editor numbers a');
lines.push('// module\'s models by their position in these lists, and that position is');
lines.push('// the index a preset stores in its slot records - so these tables are what');
lines.push('// turns a model name into the number the device loads.');
lines.push('package mooer');
lines.push('');
lines.push('// ge100ProAmpNames lists the amp models in wire index order.');
lines.push('var ge100ProAmpNames = []string{');
for (const name of amps) lines.push(`\t${JSON.stringify(name)},`);
lines.push('}');
lines.push('');
lines.push('// ge100ProCabNames lists the cabinet models in wire index order.');
lines.push('var ge100ProCabNames = []string{');
for (const name of cabs) lines.push(`\t${JSON.stringify(name)},`);
lines.push('}');
lines.push('');
lines.push('// ge100ProEffect is one model of a module: its display name, the hardware');
lines.push('// the editor says it emulates, and the model\'s knob labels in wire order.');
lines.push('type ge100ProEffect struct {');
lines.push('\tName       string');
lines.push('\tInspiredBy string');
lines.push('\tParams     []string');
lines.push('}');
lines.push('');
lines.push('// ge100ProEffects is each module\'s model list in wire index order.');
lines.push('var ge100ProEffects = map[string][]ge100ProEffect{');
for (const module of ['fx', 'od', 'ns', 'eq', 'mod', 'delay', 'reverb']) {
  const entries = effects.get(module) || [];
  if (entries.length === 0) continue;
  lines.push(`\t${JSON.stringify(module)}: {`);
  for (const e of entries) {
    const params = e.params.map((p) => JSON.stringify(p)).join(', ');
    lines.push(`\t\t{Name: ${JSON.stringify(e.name)}, InspiredBy: ${JSON.stringify(e.reference)}, Params: []string{${params}}},`);
  }
  lines.push('\t},');
}
lines.push('}');
lines.push('');

fs.writeFileSync(outPath, lines.join('\n'));

// Keep the generated file gofmt-clean so regenerating never dirties the tree.
try {
  require('child_process').execFileSync('gofmt', ['-w', outPath]);
} catch {
  console.error('warning: gofmt not available; run gofmt -w on the output');
}

console.log(
  `wrote ${amps.length} amps, ${cabs.length} cabs and ` +
    `${[...effects.values()].reduce((n, v) => n + v.length, 0)} effect models to ${outPath}`,
);
