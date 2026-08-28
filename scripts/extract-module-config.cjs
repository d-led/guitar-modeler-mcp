// Extracts the HeadRush Gigboard parameter metadata (ranges, toggles, enums)
// from headrush-desktop's renderer/config/modules/*.ts into a JSON file.
// Usage: node extract-module-config.cjs <headrush-desktop-root> <out.json>
const fs = require('fs');
const path = require('path');

const headrushRoot = process.argv[2];
const outFile = process.argv[3];
if (!headrushRoot || !outFile) {
  console.error('usage: node extract-module-config.cjs <headrush-desktop-root> <out.json>');
  process.exit(2);
}

const ts = require(path.join(headrushRoot, 'node_modules', 'typescript'));
const modulesDir = path.join(headrushRoot, 'renderer', 'config', 'modules');
const cache = new Map();

function loadModule(absPath) {
  absPath = path.resolve(absPath);
  if (cache.has(absPath)) return cache.get(absPath);

  let src = fs.readFileSync(absPath, 'utf8');
  // Strip type-only imports (named imports from the parent index or '.'),
  // which have no runtime value and would otherwise break require().
  src = src.replace(/^[ \t]*import\s*\{[^}]*\}\s*from\s*['"]\.\.?['"][;]?[ \t]*$/gm, '');

  const js = ts.transpileModule(src, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2019 },
  }).outputText;

  const module = { exports: {} };
  const requireShim = (spec) => {
    if (!spec.startsWith('.')) {
      throw new Error('unexpected non-relative import in ' + absPath + ': ' + spec);
    }
    return loadModule(path.resolve(path.dirname(absPath), spec + '.ts'));
  };

  // eslint-disable-next-line no-new-func
  new Function('require', 'module', 'exports', js)(requireShim, module, module.exports);

  // Return the raw exports object: transpiled default imports access
  // require(...).default, so unwrapping here would break them.
  cache.set(absPath, module.exports);
  return module.exports;
}

const index = loadModule(path.join(modulesDir, 'index.ts'));
const modulesConfig = index && index.modulesConfig;
if (!modulesConfig) {
  console.error('failed to locate modulesConfig export');
  process.exit(1);
}

fs.mkdirSync(path.dirname(outFile), { recursive: true });
fs.writeFileSync(outFile, JSON.stringify(modulesConfig, null, 2) + '\n');
const count = Object.keys(modulesConfig).length;
let paramCount = 0;
for (const m of Object.values(modulesConfig)) paramCount += Object.keys(m).length;
console.error(`wrote ${outFile}: ${count} modules, ${paramCount} params`);
