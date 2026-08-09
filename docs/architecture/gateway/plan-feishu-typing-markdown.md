# Feishu Typing + Markdown Implementation Plan

> **For agentic workers:** Implement task-by-task with TDD. Checkboxes track progress.

**Goal:** Feishu replies render Markdown; show Typing reaction while Agent works (Hermes-aligned).

**Architecture:** Pure helpers for post payload; Feishu adapter sends post/reply + reactions; Runner drives processing lifecycle via optional `ProcessingIndicator`.

**Tech Stack:** Go, larksuite/oapi-sdk-go/v3 im.v1

## Global Constraints

- No interactive card approval
- Reaction permission failures must not block replies
- Prefer post+md; fallback text on invalid post

---

### Task 1: Markdown post payload helper

**Files:**
- Create: `internal/gateway/platforms/feishu/post.go`
- Create: `internal/gateway/platforms/feishu/post_test.go`

- [ ] Write tests for plain md row and fenced-code split rows
- [ ] Implement `BuildMarkdownPostPayload` / `BuildMarkdownPostRows`
- [ ] `go test ./internal/gateway/platforms/feishu/ -run Post`

### Task 2: SendText as post + reply + text fallback

**Files:**
- Modify: `internal/gateway/platforms/feishu/adapter.go`

- [ ] Prefer `post` content from helper; on invalid post response → text
- [ ] If `ReplyToID` set, use `Im.V1.Message.Reply`; else `Create`
- [ ] Keep existing Create path as fallback when reply fails with target-missing codes if easy

### Task 3: ProcessingIndicator + reactions

**Files:**
- Modify: `internal/gateway/adapter.go`
- Create: `internal/gateway/platforms/feishu/reaction.go`
- Modify: `internal/gateway/platforms/feishu/adapter.go`
- Modify: `internal/gateway/runner.go`
- Modify: `internal/gateway/runner_test.go`

- [ ] Add optional `ProcessingIndicator` interface
- [ ] Feishu: create Typing, delete by reaction_id cache, MarkFailed → CrossMark
- [ ] Runner: MarkProcessing before turn; Clear on success; MarkFailed on error
- [ ] Unit test with fake adapter tracking calls

### Task 4: Docs + deploy

**Files:**
- Modify: `docs/architecture/gateway/acceptance-m1.md` / README permissions note

- [ ] Document reaction permission + post markdown behavior
- [ ] `go test ./internal/gateway/...`
- [ ] Commit/push/restart gateway on agent host (when user wants deploy)
