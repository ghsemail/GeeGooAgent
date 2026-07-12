package llm

import "context"

// MockProvider returns scripted responses for tests.
type MockProvider struct {
	ModelName string
	Responses []*Response
	Err       error
	Stream    bool // when true, ChatStream emits content rune-by-rune
}

func (m *MockProvider) Model() string {
	if m.ModelName != "" {
		return m.ModelName
	}
	return "mock-model"
}

func (m *MockProvider) Chat(ctx context.Context, messages []Message, tools []ToolSchema, temperature float64, maxTokens int) (*Response, error) {
	_ = ctx
	_ = messages
	_ = tools
	_ = temperature
	_ = maxTokens
	if m.Err != nil {
		return nil, m.Err
	}
	if len(m.Responses) == 0 {
		return &Response{Content: "mock empty", Usage: TokenUsage{Model: m.Model()}}, nil
	}
	resp := m.Responses[0]
	m.Responses = m.Responses[1:]
	return resp, nil
}

func (m *MockProvider) ChatStream(
	ctx context.Context,
	messages []Message,
	tools []ToolSchema,
	temperature float64,
	maxTokens int,
	onDelta StreamHandler,
) (*Response, error) {
	resp, err := m.Chat(ctx, messages, tools, temperature, maxTokens)
	if err != nil || resp == nil {
		return resp, err
	}
	if onDelta != nil && m.Stream && resp.Content != "" && len(resp.ToolCalls) == 0 {
		for _, r := range []rune(resp.Content) {
			onDelta(StreamDelta{Content: string(r)})
		}
	} else if onDelta != nil && resp.Content != "" && len(resp.ToolCalls) == 0 {
		onDelta(StreamDelta{Content: resp.Content})
	}
	return resp, nil
}
