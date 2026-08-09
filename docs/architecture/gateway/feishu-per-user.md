# Per-user Feishu IM Gateway

**Date:** 2026-08-09  
**Status:** Approved (scheme C / storage approach 1)

## Goal

Each ops-console user has their own Feishu App credentials; `geegoo gateway run` keeps one WebSocket per configured user. Identity comes from `X-User-Id` + stored `mcp_token` (bound on Web Gateway).

## Storage

`{outputDir}/user_gateway_feishu/{safeUserId}.json` (mode 0600)

Fields: `user_id`, `mcp_token`, `app_id`, `app_secret`, `domain`, `bot_*`, `allowed_users`, `enabled`, `updated_at`

Reload: touch `{outputDir}/user_gateway_feishu/.reload` on save; gateway polls every ~8s.

## APIs

All `/v1/gateway/feishu/*` require `X-User-Id`. Save captures `X-MCP-Token` into the user file.

## Runtime

Multi-tenant runner: sync adapters from store; inbound uses that user's `mcp_token` / `user_id` for Agent tools.

## Migration

If user has no file but host `~/.geegoo/.env` has Feishu creds, first authenticated status/setup call claims them into that user.
