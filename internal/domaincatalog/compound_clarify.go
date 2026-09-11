package domaincatalog

// CompoundStepClarifyQuestion is shown when one utterance contains both analysis and backtest.
const CompoundStepClarifyQuestion = "你希望我先做哪一步？"

// CompoundStepClarifyChoices disambiguate step priority for compound analyze+backtest requests.
var CompoundStepClarifyChoices = []string{
	"先只做分析",
	"先只做回测",
}
