# Macro JEV backfill fixtures (tests only)

- `backfill_export_it.json` — minimal 8-symbol IT loop (bond **2×** US legs only).
- `synthesis/*.json` — recorded Agent synthesis for HTTP IT; not used in production apply.
- Production-shaped **33/5 bond** export: built in-memory by `macro_jev_backfill_fixtures.build_prod_shape_audit_export` (example date **2026-11-17**).

Production backfill must supply real BSON/API export; bond manifest length comes from export, not these files.