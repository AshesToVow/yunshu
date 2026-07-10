# Yunshu — Design System

Canonical frontend tokens: `web/DESIGN.md` and `web/src/styles/`.

## Lane

Enterprise **product UI** on Ant Design 5. Preserve information density; polish spacing, type, and color discipline.

## Color

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `--ys-brand` | `#0d9488` | `#14b8a6` | Primary actions, active nav |
| `--ys-brand-strong` | `#0f766e` | `#0d9488` | Hover / emphasis |
| `--ys-bg` | `#f4f6f8` | `#141414` | App background |
| `--ys-surface` | `#ffffff` | `#1f1f1f` | Cards, header |
| `--ys-border` | `#e2e8f0` | `#303030` | Dividers |
| `--ys-text` | `#0f172a` | `rgba(255,255,255,0.88)` | Body |
| `--ys-text-muted` | `#64748b` | `rgba(255,255,255,0.55)` | Secondary |

Avoid purple-to-blue marketing gradients on auth screens. Tint neutrals (slate), never pure `#000` / `#fff` text on saturated fills.

## Typography

- Sans: `"IBM Plex Sans", "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif`
- Mono: `"JetBrains Mono", "IBM Plex Mono", Consolas, monospace`
- Page title: 15–16px / 600
- Body / table: 13px

## Spacing

`4 / 8 / 12 / 16 / 24 / 32` px

## Motion

- UI transitions: 150–200ms ease-out
- Respect `prefers-reduced-motion: reduce`

## Auth

- Split layout; default accent teal (`emerald`)
- No emoji; no AI-slop purple gradients

## Skills

- `.agents/skills/design-taste-frontend` — login / brand anti-slop
- `.agents/skills/impeccable` — product UI polish & audit
