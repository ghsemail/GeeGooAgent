# Session Store Index Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade GeeGooAgent chat sessions from plain JSON files into a searchable file-backed Session Store with structured metadata and an index, without changing deterministic workflow behavior.

**Architecture:** Keep `internal/infra.StateStore` as the persistence primitive. Extend `internal/chatsession` with metadata extraction and an index document under the existing state root, so later SQLite support can replace the backing implementation without changing runtime callers.

**Tech Stack:** Go standard library, existing `infra.StateStore`, existing `chatsession.ChatSessionStore`, `go test`.

---

### File Structure

- Modify `internal/chatsession/store.go`: add session metadata fields, save/load compatibility, index update on save, list/search helpers.
- Modify `internal/chatsession/recall.go`: prefer indexed candidates for past-session recall while preserving fallback behavior.
- Modify `internal/chatsession/store_test.go`: add TDD coverage for index creation, metadata extraction, filtering, corrupt index fallback, and legacy compatibility.
- No change to `internal/workflow/*`: deterministic workflow engine remains untouched.
- No change to `internal/infra/state.go` unless a small helper is absolutely necessary.

### Task 1: Session Metadata and Index Persistence

**Files:**
- Modify: `internal/chatsession/store_test.go`
- Modify: `internal/chatsession/store.go`

- [ ] **Step 1: Write failing tests**

Add tests that create a session with user text and tool calls, save it, then assert that the store can list indexed sessions with summary fields, tool names, and symbols.

Run: `go test ./internal/chatsession -run TestChatSessionStoreIndexesMetadata -v`
Expected: FAIL because index APIs and metadata fields do not exist yet.

- [ ] **Step 2: Implement minimal metadata/index code**

Add `ChatSessionIndexEntry`, `ChatSessionIndex`, `ListIndexedSessions`, and automatic index refresh in `Save`.

- [ ] **Step 3: Verify task test passes**

Run: `go test ./internal/chatsession -run TestChatSessionStoreIndexesMetadata -v`
Expected: PASS.

### Task 2: Index Resilience and Legacy Compatibility

**Files:**
- Modify: `internal/chatsession/store_test.go`
- Modify: `internal/chatsession/store.go`

- [ ] **Step 1: Write failing tests**

Add tests showing old session JSON without metadata still loads, and a missing/corrupt index can be rebuilt from `chat/{id}.json` files.

Run: `go test ./internal/chatsession -run 'TestChatSessionStore(LoadsLegacySession|RebuildsIndex)' -v`
Expected: FAIL until rebuild logic exists.

- [ ] **Step 2: Implement rebuild fallback**

When loading the index fails or is absent, scan existing chat sessions and derive entries from persisted messages.

- [ ] **Step 3: Verify task tests pass**

Run: `go test ./internal/chatsession -run 'TestChatSessionStore(LoadsLegacySession|RebuildsIndex)' -v`
Expected: PASS.

### Task 3: Recall Uses Indexed Candidates

**Files:**
- Modify: `internal/chatsession/store_test.go`
- Modify: `internal/chatsession/recall.go`
- Modify: `internal/chatsession/store.go`

- [ ] **Step 1: Write failing test**

Add a test that creates multiple sessions and verifies recall can find a matching stock/tool session through indexed metadata while excluding the current session.

Run: `go test ./internal/chatsession -run TestSearchPastSessionsUsesIndexMetadata -v`
Expected: FAIL until recall consumes indexed candidates.

- [ ] **Step 2: Implement indexed candidate selection**

Add a small store helper that returns recent candidate IDs from index entries. Keep full session loading for final snippet/event extraction, preserving existing result shape.

- [ ] **Step 3: Verify task test passes**

Run: `go test ./internal/chatsession -run TestSearchPastSessionsUsesIndexMetadata -v`
Expected: PASS.

### Task 4: Regression and Full Verification

**Files:**
- No planned production edits.

- [ ] **Step 1: Run package tests**

Run: `go test ./internal/chatsession -v`
Expected: PASS.

- [ ] **Step 2: Run full repository tests**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 3: Inspect git diff**

Run: `git diff -- internal/chatsession docs/superpowers/plans/2026-07-01-session-store-index.md`
Expected: diff only contains Phase 1 session-index plan and implementation.
