package llm

import "context"

// TaskKind classifies why the Kernel is calling the model.
type TaskKind string

const (
	TaskChat      TaskKind = "chat"
	TaskCompress  TaskKind = "compress"
	TaskSynthesis TaskKind = "synthesis"
	TaskComplex   TaskKind = "complex"
)

// Request is the input to Model Policy (strategy layer above Gateway).
type Request struct {
	Kind              TaskKind
	ToolSchemaCount   int
	ThinkingEnabled   bool
	Round             int
	MaxRounds         int
}

// Decision is the policy output consumed by Gateway / callers.
type Decision struct {
	Temperature    float64
	MaxTokens      int
	PreferCompress bool
	ModelHint      string // empty = keep gateway primary model
}

// Policy selects temperature / token budget / hints without owning providers.
type Policy interface {
	Decide(req Request) Decision
}

// ConfigPolicyInput maps config.json llm fields into a static policy.
type ConfigPolicyInput struct {
	Temperature         float64
	MaxTokens           int
	CompressTemperature float64
	CompressMaxTokens   int
}

// ConfigPolicy is the default Go policy: chat/synthesis share config; compress has dedicated knobs.
type ConfigPolicy struct {
	chatTemp       float64
	chatMaxTokens  int
	compressTemp   float64
	compressMaxTok int
}

// NewConfigPolicy builds a behavior-preserving policy from config-like fields.
func NewConfigPolicy(in ConfigPolicyInput) ConfigPolicy {
	temp := in.Temperature
	if temp == 0 {
		temp = 0.2
	}
	maxTok := in.MaxTokens
	if maxTok == 0 {
		maxTok = 4096
	}
	cTemp := in.CompressTemperature
	if cTemp == 0 {
		cTemp = 0.2
	}
	cMax := in.CompressMaxTokens
	if cMax == 0 {
		cMax = maxTok
	}
	return ConfigPolicy{
		chatTemp: temp, chatMaxTokens: maxTok,
		compressTemp: cTemp, compressMaxTok: cMax,
	}
}

// Decide implements Policy.
func (p ConfigPolicy) Decide(req Request) Decision {
	switch req.Kind {
	case TaskCompress:
		return Decision{
			Temperature:    p.compressTemp,
			MaxTokens:      p.compressMaxTok,
			PreferCompress: true,
		}
	default:
		return Decision{Temperature: p.chatTemp, MaxTokens: p.chatMaxTokens}
	}
}

// ComplexityPolicy wraps an inner policy and raises MaxTokens for complex work.
type ComplexityPolicy struct {
	Inner               Policy
	ComplexMinTokens    int
	ToolSchemaThreshold int
}

// Decide implements Policy.
func (p ComplexityPolicy) Decide(req Request) Decision {
	inner := p.Inner
	if inner == nil {
		inner = NewConfigPolicy(ConfigPolicyInput{})
	}
	d := inner.Decide(req)
	minTok := p.ComplexMinTokens
	if minTok <= 0 {
		minTok = 8192
	}
	threshold := p.ToolSchemaThreshold
	if threshold <= 0 {
		threshold = 40
	}
	if req.Kind == TaskComplex || req.ToolSchemaCount >= threshold {
		if d.MaxTokens < minTok {
			d.MaxTokens = minTok
		}
	}
	return d
}

type callMetaKey struct{}

// CallMeta travels on context so Gateway can apply Policy without expanding every signature.
type CallMeta struct {
	Kind            TaskKind
	ToolSchemaCount int
	ThinkingEnabled bool
	Round           int
	MaxRounds       int
}

// WithCallMeta attaches model-policy metadata to ctx.
func WithCallMeta(ctx context.Context, meta CallMeta) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, callMetaKey{}, meta)
}

// CallMetaFrom returns metadata previously attached with WithCallMeta.
func CallMetaFrom(ctx context.Context) CallMeta {
	if ctx == nil {
		return CallMeta{}
	}
	if v, ok := ctx.Value(callMetaKey{}).(CallMeta); ok {
		return v
	}
	return CallMeta{}
}

// RequestFromCallMeta maps context metadata into a Policy Request.
func RequestFromCallMeta(meta CallMeta) Request {
	kind := meta.Kind
	if kind == "" {
		kind = TaskChat
	}
	return Request{
		Kind:            kind,
		ToolSchemaCount: meta.ToolSchemaCount,
		ThinkingEnabled: meta.ThinkingEnabled,
		Round:           meta.Round,
		MaxRounds:       meta.MaxRounds,
	}
}
