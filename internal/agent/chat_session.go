package agent

import (
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

// RuntimeSessionFromChat reconstructs loop state from a persisted chat session.
func RuntimeSessionFromChat(chat *chatsession.ChatSession) *runtime.Session {
	if chat == nil {
		return runtime.NewSession()
	}
	parentID, lineageRoot, generation := chat.LineageFromMetadata()
	session := &runtime.Session{
		ID:                   chat.ID,
		Messages:             chat.RuntimeMessages(),
		StepCounter:          chat.StepCounter,
		CreatedAt:            chat.CreatedAt,
		ParentID:             parentID,
		LineageRoot:          lineageRoot,
		CompactionGeneration: generation,
		LineageChain:         chat.LineageChainFromMetadata(),
	}
	if step, calls, ok := chat.HeldPlanFromMetadata(); ok {
		session.PendingPlan = &runtime.PendingPlan{Step: step, ToolCalls: calls}
	}
	return session
}

// SyncChatFromRuntime copies loop session fields back into the chat session.
func SyncChatFromRuntime(chat *chatsession.ChatSession, rt *runtime.Session, newRecords []chatsession.ChatStepRecord) {
	if chat == nil || rt == nil {
		return
	}
	chat.SyncFromRuntime(rt.Messages, rt.StepCounter, newRecords)
	chat.SyncLineageFromRuntime(rt.ParentID, rt.LineageRoot, rt.CompactionGeneration)
	chat.SyncLineageChain(rt.LineageChain)
	if rt.PendingPlan != nil && len(rt.PendingPlan.ToolCalls) > 0 {
		chat.SyncHeldPlan(rt.PendingPlan.Step, rt.PendingPlan.ToolCalls)
	} else {
		chat.SyncHeldPlan(0, nil)
	}
}
