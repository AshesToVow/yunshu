/**
 * 业务页/组件保留 @ts-nocheck，Pro 壳层走严格 TS。
 */
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const SRC = path.resolve(__dirname, '../src');

const STRICT_DIRS = new Set([
  'pages/dynamic-menu',
  'pages/user',
  'pages/workflow/inbox',
  'pages/core',
  'pages/exception',
  'services/yunshu',
  'utils/yunshu-menu.tsx',
  'constants/brand.ts',
  'components/LegacyShell.tsx',
  'app.tsx',
  'global.tsx',
  'requestErrorConfig.ts',
  'access.ts',
]);

function isStrict(rel) {
  const r = rel.replace(/\\/g, '/');
  for (const s of STRICT_DIRS) {
    if (r === s || r.startsWith(`${s}/`)) return true;
  }
  return false;
}

function stamp(dir, rel = '') {
  for (const name of fs.readdirSync(dir)) {
    const full = path.join(dir, name);
    const relPath = rel ? `${rel}/${name}` : name;
    if (fs.statSync(full).isDirectory()) {
      if (name === '.umi' || name === '.umi-production') continue;
      stamp(full, relPath);
      continue;
    }
    if (!/\.tsx?$/.test(name)) continue;
    if (isStrict(relPath)) continue;
    let src = fs.readFileSync(full, 'utf8');
    if (src.startsWith('// @ts-nocheck')) continue;
    fs.writeFileSync(full, `// @ts-nocheck\n${src}`);
  }
}

stamp(SRC);
console.log('Stamped @ts-nocheck on business sources.');
