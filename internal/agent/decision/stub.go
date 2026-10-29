package decision

import "context"

// StubHook returns a fixed choice for tests (who=jev).
type StubHook struct {
	ChoiceID   string
	Confidence float64
	Err        error
}

func (s StubHook) Decide(_ context.Context, q Question) (Answer, error) {
	if s.Err != nil {
		return Answer{}, s.Err
	}
	id := s.ChoiceID
	if id == "" && len(q.Choices) > 0 {
		id = q.Choices[0].ID
	}
	return Answer{
		QuestionID: q.ID,
		ChoiceID:   id,
		Confidence: s.Confidence,
		Who:        WhoJev,
		Reason:     "stub",
	}, nil
}
