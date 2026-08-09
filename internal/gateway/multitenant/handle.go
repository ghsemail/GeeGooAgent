package multitenant

import (
	"context"
	"log/slog"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/gateway"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func (tr *Runner) handleOwned(ctx context.Context, ev gateway.InboundEvent, own ownedInbound) error {
	if ev.MessageID != "" && !tr.seen.TryAdd(ev.MessageID) {
		return nil
	}
	if !own.AllowAll {
		if len(own.Allowed) > 0 {
			if _, ok := own.Allowed[ev.UserID]; !ok {
				slog.Debug("gateway: feishu user not in tenant allowlist", "owner", own.OwnerUserID, "feishu_user", ev.UserID)
				return nil
			}
		}
	}
	text := trimText(ev.Text)
	if text == "" {
		return nil
	}

	key := gateway.SessionKey(ev.Platform, ev.ChatID, ev.UserID)
	lock := tr.lockFor(key)
	lock.Lock()
	defer lock.Unlock()

	adapter := own.Adapter
	if adapter == nil {
		return errString("nil adapter")
	}

	if ind, ok := adapter.(gateway.ProcessingIndicator); ok && ev.MessageID != "" {
		_ = ind.MarkProcessing(ctx, ev.MessageID)
	}

	reply, err := tr.runOwnedTurn(ctx, key, ev, text, own)
	if err != nil {
		slog.Error("gateway: agent turn failed", "err", err, "owner", own.OwnerUserID)
		if ind, ok := adapter.(gateway.ProcessingIndicator); ok && ev.MessageID != "" {
			_ = ind.MarkFailed(ctx, ev.MessageID)
		}
		_ = adapter.SendText(ctx, gateway.OutboundText{ChatID: ev.ChatID, Text: "抱歉，处理消息时出错了。", ReplyToID: ev.MessageID})
		return err
	}
	if reply == "" {
		if ind, ok := adapter.(gateway.ProcessingIndicator); ok && ev.MessageID != "" {
			_ = ind.ClearProcessing(ctx, ev.MessageID)
		}
		return nil
	}
	if tr.DryRun {
		slog.Info("gateway: dry-run reply", "owner", own.OwnerUserID, "text", truncate(reply, 200))
		if ind, ok := adapter.(gateway.ProcessingIndicator); ok && ev.MessageID != "" {
			_ = ind.ClearProcessing(ctx, ev.MessageID)
		}
		return nil
	}
	sendErr := adapter.SendText(ctx, gateway.OutboundText{ChatID: ev.ChatID, Text: reply, ReplyToID: ev.MessageID})
	if ind, ok := adapter.(gateway.ProcessingIndicator); ok && ev.MessageID != "" {
		if sendErr != nil {
			_ = ind.MarkFailed(ctx, ev.MessageID)
		} else {
			_ = ind.ClearProcessing(ctx, ev.MessageID)
		}
	}
	return sendErr
}

func (tr *Runner) runOwnedTurn(ctx context.Context, key string, ev gateway.InboundEvent, text string, own ownedInbound) (string, error) {
	store, err := tr.App.SessionStore()
	if err != nil {
		return "", err
	}
	chat, err := tr.loadOrCreateChat(store, key, ev, own.OwnerUserID)
	if err != nil {
		return "", err
	}
	rt := agent.RuntimeSessionFromChat(chat)
	chat.SyncChatSystemPrompt()
	rt.Messages = chat.RuntimeMessages()

	toolNames := tools.RegisteredChatToolNamesFor(tr.App.Registry, tr.App.Config.EffectiveChatToolsets())
	schemas := tr.App.Registry.Schemas(toolNames)
	toolCtx := tr.App.ToolContext(rt.ID)
	toolCtx.DryRun = tr.DryRun || (tr.App.Config != nil && tr.App.Config.DryRun)
	toolCtx.Interactive = false
	toolCtx.UserID = own.OwnerUserID
	if tok := trimText(own.MCPToken); tok != "" {
		toolCtx.MCPToken = tok
	}

	tr.agentMu.Lock()
	defer tr.agentMu.Unlock()

	var progress *progressPublisher
	if own.ToolProgress && !tr.DryRun {
		if ed, ok := own.Adapter.(gateway.EditableMessenger); ok {
			progress = newProgressPublisher(ctx, ed, ev.ChatID, ev.MessageID)
			tr.App.Agent.SetProgress(progress.OnEvent)
			defer tr.App.Agent.SetProgress(nil)
			defer progress.Flush()
		}
	}

	result := tr.App.Agent.Run(ctx, rt, text, toolCtx, schemas)
	newRecords := make([]chatsession.ChatStepRecord, 0, len(result.StepRecords))
	for _, rec := range result.StepRecords {
		newRecords = append(newRecords, chatsession.ChatStepRecord{
			Step: rec.Step, Timestamp: rec.Timestamp, Kind: rec.Kind,
			ToolName: rec.ToolName, ToolStatus: rec.ToolStatus, Summary: rec.Summary,
		})
	}
	agent.SyncChatFromRuntime(chat, rt, newRecords)
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	chat.Metadata["gateway_platform"] = string(ev.Platform)
	chat.Metadata["gateway_key"] = key
	chat.Metadata["gateway_chat_id"] = ev.ChatID
	chat.Metadata["gateway_feishu_user"] = ev.UserID
	chat.Metadata["gateway_owner_user_id"] = own.OwnerUserID
	chatsession.SetUserID(chat, own.OwnerUserID)
	if err := store.Save(chat); err != nil {
		slog.Warn("gateway: save session", "err", err)
	}
	if result.Failed && result.AssistantText == "" {
		return "", errString("agent turn failed")
	}
	return result.AssistantText, nil
}

func (tr *Runner) loadOrCreateChat(store chatsession.SessionStore, key string, ev gateway.InboundEvent, ownerUserID string) (*chatsession.ChatSession, error) {
	if id, ok := tr.Sessions.Get(key); ok && id != "" {
		chat, err := store.Load(id)
		if err != nil {
			return nil, err
		}
		if chat != nil {
			return chat, nil
		}
	}
	chat, err := store.Create()
	if err != nil {
		return nil, err
	}
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	chat.Metadata["gateway_platform"] = string(ev.Platform)
	chat.Metadata["gateway_key"] = key
	chat.Metadata["gateway_owner_user_id"] = ownerUserID
	chatsession.SetUserID(chat, ownerUserID)
	if err := store.Save(chat); err != nil {
		return nil, err
	}
	_ = tr.Sessions.Put(key, chat.ID)
	return chat, nil
}

func trimText(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 {
		c := s[len(s)-1]
		if c == ' ' || c == '\n' || c == '\t' || c == '\r' {
			s = s[:len(s)-1]
			continue
		}
		break
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
