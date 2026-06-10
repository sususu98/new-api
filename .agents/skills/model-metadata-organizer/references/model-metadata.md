# Model Metadata Reference

## Scope

This reference covers model catalog metadata in new-api:

- `models`: per-model or rule-based metadata.
- `vendors`: provider names and Lobe icon names.
- `abilities`: enabled model names by group/channel.
- `/api/pricing`: public pricing catalog output.

Use this when fixing wrong provider icons, vendor filters, model family labels,
or prefix rules.

## Name Rules

Defined in `model/model_meta.go`:

| Value | Constant           | Match               |
| ----: | ------------------ | ------------------- |
|     0 | `NameRuleExact`    | exact model name    |
|     1 | `NameRulePrefix`   | `strings.HasPrefix` |
|     2 | `NameRuleContains` | `strings.Contains`  |
|     3 | `NameRuleSuffix`   | `strings.HasSuffix` |

`model/pricing.go` builds `metaMap` in this order:

1. Exact metadata rows.
2. Prefix rules, only if no exact metadata exists for that enabled model.
3. Suffix rules, only if still unmatched.
4. Contains rules, only if still unmatched.
5. Default vendor inference only for models with no metadata match.

Important: an empty prefix row such as `claude-` with `vendor_id=0` blocks
default inference for all matching Claude models. Fill the row instead of
expecting fallback inference to work.

## Tables

Useful columns:

```text
models:
  id, model_name, description, icon, tags, vendor_id, endpoints, status,
  sync_official, created_time, updated_time, deleted_at, name_rule

vendors:
  id, name, description, icon, status, created_time, updated_time, deleted_at

abilities:
  group, model, channel_id, enabled, priority, weight, tag
```

## Discovery Queries

List enabled models for a suspected family:

```bash
docker exec new-api-postgres psql -U root -d new-api -Atc \
  "select distinct model
     from abilities
    where enabled = true
      and lower(model) like '%minimax%'
    order by model;"
```

Inspect current model/vendor rows:

```bash
docker exec new-api-postgres psql -U root -d new-api -Atc \
  "select id,model_name,description,icon,tags,vendor_id,endpoints,status,sync_official,name_rule
     from models
    where model_name in ('claude-','gemini-','gpt-','codex-','MiniMax-')
    order by id;
   select 'VENDORS';
   select id,name,description,icon,status
     from vendors
    order by id;"
```

Check available Lobe icon exports:

```bash
rg -n "Claude|Gemini|OpenAI|Codex|Minimax|MiniMax" \
  web/default/node_modules/@lobehub/icons/es \
  web/default/node_modules/@lobehub/icons/README.md
```

For exact compounded members, inspect the icon type file:

```bash
sed -n '1,120p' web/default/node_modules/@lobehub/icons/es/Minimax/index.d.ts
```

## Known Provider Metadata

Use the exact icon strings supported by the installed `@lobehub/icons` package.

| Family prefix | Vendor      | Model icon      | Vendor icon     | Tags               |
| ------------- | ----------- | --------------- | --------------- | ------------------ |
| `claude-`     | `Anthropic` | `Claude.Color`  | `Claude.Color`  | `anthropic,claude` |
| `gemini-`     | `Google`    | `Gemini.Color`  | `Gemini.Color`  | `google,gemini`    |
| `gpt-`        | `OpenAI`    | `OpenAI`        | `OpenAI`        | `openai,gpt`       |
| `codex-`      | `OpenAI`    | `Codex.Color`   | `OpenAI`        | `openai,codex`     |
| `MiniMax-`    | `MiniMax`   | `Minimax.Color` | `Minimax.Color` | `minimax`          |
| `glm-`        | `智谱`      | `Zhipu.Color`   | `Zhipu.Color`   | `zhipu,glm`        |
| `spark-`      | `讯飞`      | `Spark.Color`   | `Spark.Color`   | `xunfei,spark`     |
| `grok-`       | `xAI`       | `XAI`           | `XAI`           | `xai,grok`         |

Note the casing: the Lobe icon component is `Minimax`, so the color icon is
`Minimax.Color`, not `MiniMax.Color`.

## Backup Pattern

Always backup before changing metadata:

```bash
mkdir -p data/backups

docker exec new-api-postgres psql -U root -d new-api -AtF $'\t' -c \
  "select id,model_name,description,icon,tags,vendor_id,endpoints,status,
          sync_official,created_time,updated_time,deleted_at,name_rule
     from models
    where model_name in ('gemini-','gpt-','codex-','MiniMax-')
       or model_name like 'MiniMax-%'
    order by id;" \
  > "data/backups/models-vendor-icons-before-$(date +%Y%m%d-%H%M%S).tsv"

docker exec new-api-postgres psql -U root -d new-api -AtF $'\t' -c \
  "select id,name,description,icon,status,created_time,updated_time,deleted_at
     from vendors
    order by id;" \
  > "data/backups/vendors-before-$(date +%Y%m%d-%H%M%S).tsv"
```

## Update Pattern

Use a transaction or `DO` block. Preserve existing endpoint rules unless the
user asked to change endpoint support.

```bash
docker exec new-api-postgres psql -U root -d new-api -v ON_ERROR_STOP=1 -Atc "
DO \$\$
DECLARE
  now_ts bigint := floor(extract(epoch from now()))::bigint;
  minimax_id bigint;
BEGIN
  UPDATE vendors
     SET icon = 'Minimax.Color', status = 1, updated_time = now_ts
   WHERE name = 'MiniMax' AND deleted_at IS NULL;

  IF NOT FOUND THEN
    INSERT INTO vendors (name, description, icon, status, created_time, updated_time)
    VALUES ('MiniMax', '', 'Minimax.Color', 1, now_ts, now_ts);
  END IF;

  SELECT id INTO minimax_id
    FROM vendors
   WHERE name = 'MiniMax' AND deleted_at IS NULL
   ORDER BY id
   LIMIT 1;

  UPDATE models
     SET description = 'Google Gemini model family',
         icon = 'Gemini.Color',
         tags = 'google,gemini',
         vendor_id = 2,
         updated_time = now_ts
   WHERE model_name = 'gemini-' AND name_rule = 1 AND deleted_at IS NULL;

  UPDATE models
     SET description = 'OpenAI GPT model family',
         icon = 'OpenAI',
         tags = 'openai,gpt',
         vendor_id = 3,
         updated_time = now_ts
   WHERE model_name = 'gpt-' AND name_rule = 1 AND deleted_at IS NULL;

  UPDATE models
     SET description = 'OpenAI Codex model family',
         icon = 'Codex.Color',
         tags = 'openai,codex',
         vendor_id = 3,
         updated_time = now_ts
   WHERE model_name = 'codex-' AND name_rule = 1 AND deleted_at IS NULL;

  UPDATE models
     SET description = 'MiniMax model family',
         icon = 'Minimax.Color',
         tags = 'minimax',
         vendor_id = minimax_id,
         status = 1,
         sync_official = 0,
         name_rule = 1,
         updated_time = now_ts
   WHERE model_name = 'MiniMax-' AND deleted_at IS NULL;

  IF NOT FOUND THEN
    INSERT INTO models
      (model_name, description, icon, tags, vendor_id, endpoints, status,
       sync_official, created_time, updated_time, name_rule)
    VALUES
      ('MiniMax-', 'MiniMax model family', 'Minimax.Color', 'minimax',
       minimax_id, '', 1, 0, now_ts, now_ts, 1);
  END IF;
END
\$\$;"
```

## Verification

Direct DB verification:

```bash
docker exec new-api-postgres psql -U root -d new-api -Atc \
  "select m.id,m.model_name,m.icon,m.tags,m.vendor_id,v.name,v.icon,m.name_rule
     from models m
     left join vendors v on v.id = m.vendor_id
    where m.model_name in ('claude-','gemini-','gpt-','codex-','MiniMax-')
    order by m.id;"
```

Pricing API verification:

```bash
sleep 70
curl -sS http://127.0.0.1:3000/api/pricing | jq -r \
  '.data[]
   | select((.model_name|startswith("claude-"))
         or (.model_name|startswith("gemini-"))
         or (.model_name|startswith("gpt-"))
         or (.model_name|startswith("codex-"))
         or (.model_name|startswith("MiniMax-")))
   | [.model_name,.icon,(.vendor_id|tostring)] | @tsv' \
  | sort
```

Vendor API payload verification:

```bash
curl -sS http://127.0.0.1:3000/api/pricing | jq -r \
  '.vendors[]
   | select(.name=="MiniMax" or .name=="Google" or .name=="OpenAI" or .name=="Anthropic")
   | [.id,.name,.icon] | @tsv' \
  | sort -n
```

`model.GetPricing()` has a one-minute cache. Direct DB writes do not call
`model.RefreshPricing()`, so wait for natural expiration unless the user has
explicitly approved a restart/reload or a code path that refreshes cache.

## Frontend Display Notes

The model management table prefers:

```text
model.icon -> vendor icon -> first model-name letter
```

The pricing catalog maps `vendor_id` to `vendor_name`, `vendor_icon`, and
`vendor_description` in `use-pricing-data.ts`. If a component only renders
`vendor_icon`, setting `models.icon` alone is not enough for that UI. Make sure
the vendor association is correct.

If a model-specific icon should differ from the vendor icon, the backend already
returns `pricing.icon`; the frontend component may need to prefer `model.icon`
over `vendor_icon`.
