package llm

// MockProvider returns scripted responses for tests.
type MockProvider struct {
	ModelName string
	Responses []*Response
}

func (m *MockProvider) Model() string {
	if m.ModelName != "" {
		return m.ModelName
	}
	return "mock-model"
}

func (m *MockProvider) Chat(messages []Message, tools []ToolSchema, temperature float64, maxTokens int) (*Response, error) {
	if len(m.Responses) == 0 {
		return &Response{Content: "mock empty", Usage: TokenUsage{Model: m.Model()}}, nil
	}
	resp := m.Responses[0]
	m.Responses = m.Responses[1:]
	return resp, nil
}
