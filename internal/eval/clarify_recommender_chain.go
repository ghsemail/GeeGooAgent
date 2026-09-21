package eval

import "context"

// ClarifyRecommenderChain tries recommenders in order (JEV → background LLM).
type ClarifyRecommenderChain []ClarifyRecommender

func (c ClarifyRecommenderChain) Recommend(
	ctx context.Context,
	question string,
	choices []string,
	hint ClarifyRecommendContext,
) (ClarifyRecommendation, error) {
	var lastErr error
	for _, r := range c {
		if r == nil {
			continue
		}
		rec, err := r.Recommend(ctx, question, choices, hint)
		if err != nil {
			lastErr = err
			continue
		}
		if rec.valid(choices) {
			return rec, nil
		}
	}
	if lastErr != nil {
		return ClarifyRecommendation{}, lastErr
	}
	return ClarifyRecommendation{}, errClarifyRecommenderExhausted
}

var errClarifyRecommenderExhausted = clarifierExhausted{}

type clarifierExhausted struct{}

func (clarifierExhausted) Error() string { return "clarify recommender chain exhausted" }
