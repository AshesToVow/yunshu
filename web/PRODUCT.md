# Yunshu Web — Product Context

## Register

product

## Surface

**Product** (not marketing): enterprise operations console for DevOps / SRE / platform teams.

## Audience

- Internal operators managing Kubernetes, CMDB, CI/CD, alerts, and projects
- Needs clarity under pressure, not decorative marketing chrome

## Voice

- Professional, direct, operational Chinese (primary) with English labels where needed
- No emoji in UI copy
- Prefer verbs: 发布、同步、探测、授权

## Jobs to be done

1. Sign in securely (account + email OTP)
2. Navigate large menu trees (K8s, projects, system)
3. Scan dense tables and status tags quickly
4. Run destructive actions with clear affordances

## Anti-references

- Generic AI SaaS landing (purple mesh, centered hero, three feature cards)
- Industrial CRT / scanline gimmicks in daily ops views
- Playful consumer app tone

## Stack

- React 18 + Vite + TypeScript
- Ant Design 5
- CSS design tokens in `src/styles/`
