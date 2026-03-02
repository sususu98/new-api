---
name: model-metadata-organizer
description: >-
  Organize new-api model metadata for pricing and model catalog display. Use
  when the user asks to fix or整理 model icons, vendors/providers, model family
  prefixes, pricing catalog model metadata, Lobe icon names, or models table
  prefix rules such as claude-, gemini-, gpt-, codex-, MiniMax-, glm-, and
  similar provider families.
---

# Model Metadata Organizer

## Overview

Use this skill to normalize model metadata used by the pricing catalog and model
management UI: vendor records, model prefix rules, Lobe icon names, tags,
descriptions, and verification through `/api/pricing`.

Before making database edits or code changes, read
`references/model-metadata.md`.

## Workflow

1. Confirm the target model families from the user's wording and from enabled
   abilities. Prefer actual enabled model names over guessed prefixes.
2. Inspect existing vendor/model metadata before changing anything.
3. Check `@lobehub/icons` exports before choosing an icon string. Icon names are
   case-sensitive.
4. Prefer one `models` prefix rule per model family instead of adding many exact
   rows, unless exact metadata is intentionally different.
5. Back up affected `models` and `vendors` rows into `data/backups/` before DB
   writes.
6. Apply metadata changes with a small SQL transaction or `DO` block.
7. Do not restart or reload services unless the user explicitly asks.
8. Wait for the pricing cache to expire naturally, then verify `/api/pricing`
   returns the expected `icon` and `vendor_id`.

## Key Files

- `model/pricing.go`: builds pricing metadata and applies exact/prefix/suffix/contains rules.
- `model/pricing_default.go`: default vendor inference for models without metadata.
- `model/model_meta.go`: `models` schema and name rule constants.
- `model/vendor_meta.go`: `vendors` schema.
- `web/default/src/lib/lobe-icon.tsx`: frontend Lobe icon string parser.
- `web/default/src/features/pricing/hooks/use-pricing-data.ts`: maps `vendor_id` to vendor display fields.

## Safety Rules

- Direct DB metadata edits are acceptable for display fixes, but always create
  TSV backups first.
- Do not clear `endpoints`, `status`, `name_rule`, or `sync_official` unless the
  change requires it.
- If an empty prefix metadata row exists, fill it in; do not rely on default
  inference, because existing metadata blocks fallback inference.
- If the frontend display still looks wrong after metadata is correct, inspect
  whether the component prefers `vendor_icon` over `model.icon` before changing
  data again.
