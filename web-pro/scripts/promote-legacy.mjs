/**
 * 一次性迁移：将 src/legacy 提升为 web-pro/src 正式业务源码，并删除 legacy 目录。
 * 用法：node scripts/promote-legacy.mjs
 */
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const SRC = path.resolve(__dirname, '../src');
const LEGACY = path.join(SRC, 'legacy');

/** 已存在的 Pro 壳层文件/目录，不覆盖 */
const PROTECTED = new Set([
  'pages/dynamic-menu',
  'pages/user',
  'pages/workflow/inbox',
  'pages/core',
  'pages/exception',
  'constants/brand.ts',
  'utils/yunshu-menu.tsx',
  'utils/format.ts',
  'utils/chinaDivision.ts',
  'utils/chinaDivision.test.ts',
]);

function norm(p) {
  return p.replace(/\\/g, '/');
}

function isProtected(rel) {
  const r = norm(rel);
  for (const p of PROTECTED) {
    if (r === p || r.startsWith(`${p}/`)) return true;
  }
  return false;
}

function copyMerge(from, to, rel = '') {
  for (const name of fs.readdirSync(from)) {
    const s = path.join(from, name);
    const d = path.join(to, name);
    const relPath = rel ? `${rel}/${name}` : name;
    if (isProtected(relPath)) {
      console.log('  skip (protected)', relPath);
      continue;
    }
    const st = fs.statSync(s);
    if (st.isDirectory()) {
      fs.mkdirSync(d, { recursive: true });
      copyMerge(s, d, relPath);
    } else if (!fs.existsSync(d)) {
      fs.mkdirSync(path.dirname(d), { recursive: true });
      fs.copyFileSync(s, d);
    } else if (fs.statSync(d).isFile()) {
      fs.copyFileSync(s, d);
      console.log('  overwrite', relPath);
    }
  }
}

function stripSyncHeader(dir) {
  for (const name of fs.readdirSync(dir)) {
    const full = path.join(dir, name);
    if (fs.statSync(full).isDirectory()) {
      if (name === 'legacy' || name === '.umi' || name === '.umi-production') continue;
      stripSyncHeader(full);
      continue;
    }
    if (!/\.tsx?$/.test(name)) continue;
    let src = fs.readFileSync(full, 'utf8');
    const next = src.replace(/^\/\/ @ts-nocheck[^\n]*\n/, '');
    if (next !== src) fs.writeFileSync(full, next);
  }
}

function fixI18nLocales() {
  const appLocales = path.join(SRC, 'locales/app');
  fs.mkdirSync(appLocales, { recursive: true });
  const legacyZh = path.join(LEGACY, 'locales/zh-CN.json');
  const legacyEn = path.join(LEGACY, 'locales/en-US.json');
  if (fs.existsSync(legacyZh)) fs.copyFileSync(legacyZh, path.join(appLocales, 'zh-CN.json'));
  if (fs.existsSync(legacyEn)) fs.copyFileSync(legacyEn, path.join(appLocales, 'en-US.json'));
  const i18nFile = path.join(SRC, 'i18n/index.ts');
  if (fs.existsSync(i18nFile)) {
    let src = fs.readFileSync(i18nFile, 'utf8');
    src = src
      .replace('../locales/zh-CN.json', '../locales/app/zh-CN.json')
      .replace('../locales/en-US.json', '../locales/app/en-US.json');
    fs.writeFileSync(i18nFile, src);
  }
}

function replaceLegacyImports(dir) {
  for (const name of fs.readdirSync(dir)) {
    const full = path.join(dir, name);
    if (fs.statSync(full).isDirectory()) {
      if (name === 'legacy' || name === '.umi' || name === '.umi-production') continue;
      replaceLegacyImports(full);
      continue;
    }
    if (!/\.tsx?$/.test(name)) continue;
    let src = fs.readFileSync(full, 'utf8');
    if (!src.includes('@/legacy/') && !src.includes("from '@/legacy")) continue;
    const next = src.replace(/@\/legacy\//g, '@/');
    fs.writeFileSync(full, next);
  }
}

if (!fs.existsSync(LEGACY)) {
  console.error('src/legacy not found — already promoted?');
  process.exit(1);
}

console.log('Promote src/legacy → src/ ...');

const dirs = fs.readdirSync(LEGACY).filter((n) => fs.statSync(path.join(LEGACY, n)).isDirectory());
for (const dir of dirs) {
  if (dir === 'locales') continue;
  console.log('  merge', dir);
  copyMerge(path.join(LEGACY, dir), path.join(SRC, dir), dir);
}

// contexts/auth-context.tsx：保留 legacy 中 Pro 桥接版
const authBridge = path.join(LEGACY, 'contexts/auth-context.tsx');
if (fs.existsSync(authBridge)) {
  fs.mkdirSync(path.join(SRC, 'contexts'), { recursive: true });
  fs.copyFileSync(authBridge, path.join(SRC, 'contexts/auth-context.tsx'));
}

// constants/path-component-fallback
const fallback = path.join(LEGACY, 'constants/path-component-fallback.ts');
if (fs.existsSync(fallback)) {
  fs.mkdirSync(path.join(SRC, 'constants'), { recursive: true });
  fs.copyFileSync(fallback, path.join(SRC, 'constants/path-component-fallback.ts'));
}

fixI18nLocales();

console.log('Remove src/legacy ...');
fs.rmSync(LEGACY, { recursive: true, force: true });

console.log('Fix @/legacy imports ...');
replaceLegacyImports(SRC);

console.log('Strip sync headers ...');
stripSyncHeader(SRC);

console.log('Done. Run npm run build to verify.');
