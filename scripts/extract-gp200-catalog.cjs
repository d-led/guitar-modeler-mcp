#!/usr/bin/env node
// extract-gp200-catalog.cjs
//
// Generates internal/gp200/catalog_data.go from the Valeton GP-200 effect
// tables, which are expressed as TypeScript object literals in the community
// gp200-studio project (https://github.com/kabir0st/gp200-studio):
//
//   src/core/effectNames.ts        EFFECT_MAP: effect code -> {name, module}
//   src/core/effectDescriptions.ts EFFECT_DESCRIPTIONS: name -> real hardware
//
// The effect code -> name/module mapping was itself extracted from Valeton's
// official GP-200 algorithm.xml; the descriptions come from the
// community-compiled "Valeton GP-200 FX List V2". Both are factual data, not
// creative expression, and are reproduced here as plain Go tables with
// attribution (see NOTICE in the same package).
//
// Usage: node scripts/extract-gp200-catalog.cjs <gp200-studio-src-dir> <out.go>
'use strict';

const fs = require('fs');
const path = require('path');

const srcDir = process.argv[2];
const outPath = process.argv[3];
if (!srcDir || !outPath) {
  console.error('usage: node extract-gp200-catalog.cjs <gp200-studio-src-dir> <out.go>');
  process.exit(1);
}

const effectNames = fs.readFileSync(path.join(srcDir, 'core', 'effectNames.ts'), 'utf8');
const effectDescs = fs.readFileSync(path.join(srcDir, 'core', 'effectDescriptions.ts'), 'utf8');
const effectParams = fs.readFileSync(path.join(srcDir, 'core', 'effectParams.ts'), 'utf8');

// num formats a JS number as a Go numeric literal.
function num(v) {
  if (Number.isInteger(v)) return String(v);
  return v.toFixed(6).replace(/\.?0+$/, '');
}

// EFFECT_MAP entries: `123456: { name: 'Foo', module: 'AMP' },`
const effectRe = /^\s*(\d+):\s*\{\s*name:\s*'([^']+)',\s*module:\s*'([^']+)'\s*\}/gm;
const effects = [];
let m;
while ((m = effectRe.exec(effectNames)) !== null) {
  effects.push({ code: Number(m[1]), name: m[2], module: m[3] });
}
if (effects.length === 0) {
  console.error('no EFFECT_MAP entries parsed from effectNames.ts');
  process.exit(1);
}

// EFFECT_DESCRIPTIONS entries: `'Name': 'Desc',` or `"Name": "Desc",`
const descRe = /^\s*'([^']+)':\s*'([^']*)',?\s*(?:\/\/.*)?$/gm;
const descDqRe = /^\s*'([^']+)':\s*"([^"]*)",?\s*(?:\/\/.*)?$/gm;
const descs = new Map();
while ((m = descRe.exec(effectDescs)) !== null) {
  descs.set(m[1], m[2]);
}
while ((m = descDqRe.exec(effectDescs)) !== null) {
  descs.set(m[1], m[2]);
}

// EFFECT_PARAMS entries: one `CODE: [ ... ],` block per effect. Split on the
// top-level keys, then pull type/name/idx/default/min/max/step/options from
// each parameter line.
const paramsByCode = new Map(); // code -> [{ idx, name, kind, min, max, step, def, options }]
const paramBlockRe = /^\s*(\d+):\s*\[$/gm;
const blockStarts = [];
let bm;
while ((bm = paramBlockRe.exec(effectParams)) !== null) {
  blockStarts.push({ code: Number(bm[1]), lineStart: effectParams.lastIndexOf('\n', bm.index) + 1 });
}
for (let i = 0; i < blockStarts.length; i++) {
  const start = blockStarts[i].lineStart;
  const end = i + 1 < blockStarts.length ? blockStarts[i + 1].lineStart : effectParams.length;
  const body = effectParams.slice(start, end);
  const params = [];
  const used = new Set();
  for (const line of body.split('\n')) {
    const typeMatch = line.match(/type:\s*'(knob|switch|combox)'/);
    if (!typeMatch) continue;
    const kind = typeMatch[1];
    const nameMatch = line.match(/name:\s*'([^']+)'/);
    const idxMatch = line.match(/idx:\s*(\d+)/);
    const defMatch = line.match(/default:\s*(-?[\d.]+)/);
    if (!nameMatch || !idxMatch || !defMatch) continue;

    let idx = Number(idxMatch[1]);
    // The community-generated source occasionally reuses an index within one
    // effect (e.g. Slapback lists Sync and Trail both at idx 3). Bump the
    // collision to the next free index so the array stays positional.
    while (used.has(idx)) idx++;
    used.add(idx);

    const p = {
      idx,
      name: nameMatch[1],
      kind,
      def: Number(defMatch[1]),
      min: 0,
      max: 0,
      step: 0,
      options: [],
    };
    if (kind === 'knob') {
      const minMatch = line.match(/min:\s*(-?[\d.]+)/);
      const maxMatch = line.match(/max:\s*(-?[\d.]+)/);
      const stepMatch = line.match(/step:\s*(-?[\d.]+)/);
      if (minMatch) p.min = Number(minMatch[1]);
      if (maxMatch) p.max = Number(maxMatch[1]);
      if (stepMatch) p.step = Number(stepMatch[1]);
    } else {
      // switch/combox: collect option display names in id order.
      const optRe = /\{\s*name:\s*'([^']+)'\s*,\s*id:\s*(-?\d+)\s*\}/g;
      let o;
      const opts = [];
      while ((o = optRe.exec(line)) !== null) {
        opts[Number(o[2])] = o[1];
      }
      p.options = opts.filter((x) => x !== undefined);
    }
    params.push(p);
  }
  paramsByCode.set(blockStarts[i].code, params);
}

// Sanity: every effect should have a description; allow a few unmapped.
let missing = 0;
for (const e of effects) {
  if (!descs.has(e.name)) missing++;
}
if (missing > 0) {
  console.error(`warning: ${missing} effect(s) have no description`);
}

const lines = [];
lines.push('// Code generated by scripts/extract-gp200-catalog.cjs; DO NOT EDIT.');
lines.push('//');
lines.push('// Effect tables for the Valeton GP-200. The effect code -> name/module');
lines.push('// mapping is derived from Valeton\'s official GP-200 algorithm.xml (as');
lines.push('// tabulated by gp200-studio, https://github.com/kabir0st/gp200-studio,');
lines.push('// GPL-3.0, effectNames.ts); the "inspired by" hardware descriptions come');
lines.push('// from the community-compiled "Valeton GP-200 FX List V2" (also mirrored in');
lines.push('// valeton-gp200-referencias). Reproduced as factual data with attribution.');
lines.push('package gp200');
lines.push('');
lines.push('// slotModules is the GP-200\'s fixed 11-block signal path. Each block has a');
lines.push('// dedicated function; the block index (0..10) is the physical slot stored in');
lines.push('// a .prst file, independent of the effect loaded into it.');
lines.push('var slotModules = []string{"PRE", "WAH", "DST", "AMP", "NR", "CAB", "EQ", "MOD", "DLY", "RVB", "VOL"}');
lines.push('');
lines.push('// effect is one effect model: its 32-bit on-wire code, display name and the');
lines.push('// module catalog it is listed under in the editor.');
lines.push('type effect struct {');
lines.push('\tCode   uint32');
lines.push('\tName   string');
lines.push('\tModule string');
lines.push('}');
lines.push('');
lines.push('// effects is every effect in code order (the order they appear in the');
lines.push('// editor\'s catalogs).');
lines.push('var effects = []effect{');
for (const e of effects) {
  lines.push(`\t{Code: ${e.code}, Name: ${JSON.stringify(e.name)}, Module: ${JSON.stringify(e.module)}},`);
}
lines.push('}');
lines.push('');
lines.push('// inspiredBy maps an effect\'s display name to the real hardware it emulates');
lines.push('// (empty for models with no documented original). Keyed by name only: a few');
lines.push('// names repeat across codes (User IR, SnapTone, Dark Twin, Tube).');
lines.push('var inspiredBy = map[string]string{');
const seenNames = new Set();
for (const e of effects) {
  if (seenNames.has(e.name)) continue;
  seenNames.add(e.name);
  const d = descs.get(e.name) || '';
  lines.push(`\t${JSON.stringify(e.name)}: ${JSON.stringify(d)},`);
}
lines.push('}');
lines.push('');
lines.push('// effectParams maps an effect code to its editable parameter definitions, in');
lines.push('// index order. Kind is "knob", "switch" or "combox"; knob entries carry a');
lines.push('// min/max/step range, switch/combox entries carry the option display names.');
lines.push('var effectParams = map[uint32][]ParamDef{');
for (const e of effects) {
  const ps = paramsByCode.get(e.code) || [];
  if (ps.length === 0) {
    lines.push(`\t${e.code}: nil,`);
    continue;
  }
  const defs = ps.map((p) => {
    const fields = [`Index: ${p.idx}`, `Name: ${JSON.stringify(p.name)}`, `Kind: ${JSON.stringify(p.kind)}`];
    if (p.kind === 'knob') {
      fields.push(`Min: ${num(p.min)}`, `Max: ${num(p.max)}`, `Step: ${num(p.step)}`, `Default: ${num(p.def)}`);
    } else {
      const opts = (p.options || []).map((o) => JSON.stringify(o)).join(', ');
      fields.push(`Default: ${num(p.def)}`, `Options: []string{${opts}}`);
    }
    return `{${fields.join(', ')}}`;
  });
  lines.push(`\t${e.code}: {`);
  for (const d of defs) {
    lines.push(`\t\t${d},`);
  }
  lines.push('\t},');
}
lines.push('}');
lines.push('');

fs.writeFileSync(outPath, lines.join('\n'));

// Keep the generated Go file gofmt-clean so regeneration never dirties the
// working tree.
try {
  require('child_process').execFileSync('gofmt', ['-w', outPath]);
} catch {
  console.error('warning: gofmt not available; run gofmt -w on the output');
}

console.log(`wrote ${effects.length} effects to ${outPath} (${missing} missing descriptions)`);
