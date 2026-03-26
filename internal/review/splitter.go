package review

const charsPerToken = 4

func EstimateTokens(s string) int {
	return len(s) / charsPerToken
}
