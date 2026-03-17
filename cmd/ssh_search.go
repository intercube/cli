package cmd

import (
	"sort"
	"strings"
)

type scoredSSHTarget struct {
	Target sshTargetOption
	Score  int
}

func filterAndRankSSHTargets(options []sshTargetOption, query string) []sshTargetOption {
	tokens := sshTokenizeSearch(query)
	normalizedQuery := sshNormalizeSearchValue(query)
	if len(tokens) == 0 && normalizedQuery == "" {
		result := append([]sshTargetOption(nil), options...)
		sortSSHTargetsByTitle(result)
		return result
	}

	scored := make([]scoredSSHTarget, 0, len(options))
	for _, option := range options {
		score, ok := sshScoreTarget(option, tokens, normalizedQuery)
		if !ok {
			continue
		}

		scored = append(scored, scoredSSHTarget{Target: option, Score: score})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}

		leftTitle := strings.ToLower(scored[i].Target.Title)
		rightTitle := strings.ToLower(scored[j].Target.Title)
		if leftTitle != rightTitle {
			return leftTitle < rightTitle
		}

		return strings.ToLower(scored[i].Target.HostName) < strings.ToLower(scored[j].Target.HostName)
	})

	result := make([]sshTargetOption, 0, len(scored))
	for _, item := range scored {
		result = append(result, item.Target)
	}

	return result
}

func sshTargetMatchesInput(option sshTargetOption, input string) bool {
	if strings.TrimSpace(input) == "" {
		return true
	}

	tokens := sshTokenizeSearch(input)
	normalizedInput := sshNormalizeSearchValue(input)
	_, ok := sshScoreTarget(option, tokens, normalizedInput)
	return ok
}

func sshScoreTarget(option sshTargetOption, tokens []string, normalizedQuery string) (int, bool) {
	fields := option.SearchFields
	if len(fields) == 0 {
		fields = []string{
			sshNormalizeSearchValue(option.HostName),
			sshNormalizeSearchValue(option.ServerName),
			sshNormalizeSearchValue(option.Title),
			sshNormalizeSearchValue(option.Meta),
			sshNormalizeSearchValue(option.SitePreview),
		}
	}

	if len(fields) == 0 {
		return 0, false
	}

	score := 0
	if normalizedQuery != "" {
		for _, field := range fields {
			if field == normalizedQuery {
				score += 100
				break
			}
		}
	}

	for _, token := range tokens {
		best := 0
		for _, field := range fields {
			switch {
			case field == token:
				if best < 30 {
					best = 30
				}
			case strings.HasPrefix(field, token):
				if best < 20 {
					best = 20
				}
			case strings.Contains(field, token):
				if best < 10 {
					best = 10
				}
			}
		}

		if best == 0 {
			return 0, false
		}

		score += best
	}

	score += len(tokens)
	return score, true
}

func sshNormalizeSearchValue(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(trimmed))
	for _, char := range trimmed {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

func sshTokenizeSearch(value string) []string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return nil
	}

	rawTokens := strings.FieldsFunc(trimmed, func(char rune) bool {
		return !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9'))
	})

	tokens := make([]string, 0, len(rawTokens))
	seen := make(map[string]struct{}, len(rawTokens))
	for _, token := range rawTokens {
		normalized := sshNormalizeSearchValue(token)
		if normalized == "" {
			continue
		}

		if _, exists := seen[normalized]; exists {
			continue
		}

		seen[normalized] = struct{}{}
		tokens = append(tokens, normalized)
	}

	return tokens
}

func sortSSHTargetsByTitle(options []sshTargetOption) {
	sort.SliceStable(options, func(i, j int) bool {
		leftTitle := strings.ToLower(options[i].Title)
		rightTitle := strings.ToLower(options[j].Title)
		if leftTitle != rightTitle {
			return leftTitle < rightTitle
		}

		return strings.ToLower(options[i].HostName) < strings.ToLower(options[j].HostName)
	})
}
