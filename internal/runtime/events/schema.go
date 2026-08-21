package events

// SchemaVersion is the Thread/Turn/Item event protocol version (P1-2).
const SchemaVersion = 1

// Item types shared by CLI NDJSON and HTTP SSE.
const (
	ItemUserMessage      = "user_message"
	ItemReasoning        = "reasoning"
	ItemAssistantMessage = "assistant_message"
	ItemToolCall         = "tool_call"
	ItemToolResult       = "tool_result"
	ItemPlanProposal     = "plan_proposal"
	ItemClarifyPrompt    = "clarify_prompt"
	ItemBudgetWarning    = "budget_warning"
	ItemTurnComplete     = "turn_complete"
	ItemTurnFailed       = "turn_failed"
	ItemTurnAborted      = "turn_aborted"
	ItemStatus           = "status"
)

// Turn lifecycle events.
const (
	TurnStart    = "turn_start"
	TurnComplete = "turn_complete"
	TurnFailed   = "turn_failed"
	TurnAborted  = "turn_aborted"
)

// ItemTypeForEvent maps legacy progress event names to stable item_type values.
func ItemTypeForEvent(event string) string {
	switch event {
	case TurnStart:
		return ItemUserMessage
	case "thinking", "reasoning":
		return ItemReasoning
	case "assistant_delta", "assistant_message", "reply":
		return ItemAssistantMessage
	case "tool_start", "tool_call":
		return ItemToolCall
	case "tool_end", "tool_result":
		return ItemToolResult
	case "plan_proposed", "plan_proposal":
		return ItemPlanProposal
	case "clarify":
		return ItemClarifyPrompt
	case "budget_warning":
		return ItemBudgetWarning
	case TurnComplete, "turn_end", "done":
		return ItemTurnComplete
	case TurnFailed:
		return ItemTurnFailed
	case TurnAborted:
		return ItemTurnAborted
	case "status", "step":
		return ItemStatus
	default:
		return event
	}
}
