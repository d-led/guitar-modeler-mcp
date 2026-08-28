// Extracts the modeled-after data from the gigboardhints repo into a JSON file.
// Usage: node extract-hints.cjs <gigboardhints-root> <out.json>
const fs = require('fs');
const path = require('path');

const root = process.argv[2];
const outFile = process.argv[3];
if (!root || !outFile) {
  console.error('usage: node extract-hints.cjs <gigboardhints-root> <out.json>');
  process.exit(2);
}

function loadData(file) {
  const src = fs.readFileSync(path.join(root, 'src', 'data', file), 'utf8');
  const start = src.indexOf('[');
  const end = src.lastIndexOf(']') + 1;
  // eslint-disable-next-line no-eval
  return eval('(' + src.slice(start, end) + ')');
}

const out = {
  amps: loadData('amps.js'),
  cabs: loadData('cabs.js'),
  mics: loadData('mics.js'),
  fxs: loadData('fxs.js'),
};

fs.mkdirSync(path.dirname(outFile), { recursive: true });
fs.writeFileSync(outFile, JSON.stringify(out, null, 2) + '\n');
console.error(
  `wrote ${outFile}: ${out.amps.length} amps, ${out.cabs.length} cabs, ${out.mics.length} mics, ${out.fxs.length} fxs`,
);
