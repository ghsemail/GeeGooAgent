package semantic

import (
	"context"
	"testing"
)

type stubEmbedder struct {
	vec []float32
	err error
}

func (s stubEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return s.vec, s.err
}

func TestVectorLiteral(t *testing.T) {
	got := vectorLiteral([]float32{0.1, 0.2, 0.3})
	want := "[0.1,0.2,0.3]"
	if got != want {
		t.Fatalf("vectorLiteral = %q, want %q", got, want)
	}
}

func TestOpenAIEmbedderNilWhenNoKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GEEGOO_OPENAI_API_KEY", "")
	if NewOpenAIEmbedderFromEnv() != nil {
		t.Fatal("expected nil embedder without API key")
	}
}

func TestOpenAIEmbedderEmbedEmpty(t *testing.T) {
	e := &OpenAIEmbedder{model: "text-embedding-3-small"}
	if _, err := e.Embed(context.Background(), "  "); err != nil {
		t.Fatal(err)
	}
}
