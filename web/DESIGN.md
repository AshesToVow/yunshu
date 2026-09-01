# Yunshu Web — Design System

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
- Mono: `"JetBrains Mono", "IBM Plex Mono", Consolas, monospace` (logs, YAML, terminals)
- Page title: 15–16px / 600
- Section: 14px / 600
- Body: 13–14px / 400
- Table: 13px

## Spacing scale

`4 / 8 / 12 / 16 / 24 / 32` px — use multiples; card padding `16–20px`, content gutter `20–24px`.

## Radius

- Ops shell: `8px` (`--ops-radius`)
- Buttons/inputs: follow Ant Design with token override

## Motion

- UI transitions: `150–200ms` `ease-out`
- Respect `prefers-reduced-motion: reduce` — disable decorative background animation

## Components

- Prefer Ant Design primitives; customize via CSS variables + `ops-platform` scope
- Tables: comfortable row height, visible hover, sticky header where used
- Tags: semantic colors from `--ops-*-bg/text` tokens
- No nested card-in-card unless grouping distinct regions

## Auth (login)

- Split layout: brand story left / form right (configurable)
- Default accent: **teal** (`emerald`), not violet/blue
- Form surface: frosted panel with 1px border, solid fallback

## References

- Taste Skill: `.agents/skills/design-taste-frontend` (login redesign, anti-slop)
- Impeccable: `.agents/skills/impeccable` (polish, audit, tokens)
- Global CSS: `web/src/styles/yunshu-ops-global.css` (last-loaded overrides)
