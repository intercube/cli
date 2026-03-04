package contextconfig

import "strings"

type Inputs struct {
	FlagValue           string
	LegacyEnvValue      string
	PreferredEnvValue   string
	ContextConfigValue  string
	BehaviorDefault     string
	SessionDefaultValue string
	UserDefaultValue    string
}

type Output struct {
	Value  string
	Source string
}

func ResolveValue(input Inputs) Output {
	if value := strings.TrimSpace(input.FlagValue); value != "" {
		return Output{Value: value, Source: "flag"}
	}

	if value := strings.TrimSpace(input.PreferredEnvValue); value != "" {
		return Output{Value: value, Source: "env"}
	}

	if value := strings.TrimSpace(input.LegacyEnvValue); value != "" {
		return Output{Value: value, Source: "env"}
	}

	if value := strings.TrimSpace(input.ContextConfigValue); value != "" {
		return Output{Value: value, Source: "context"}
	}

	if value := strings.TrimSpace(input.BehaviorDefault); value != "" {
		return Output{Value: value, Source: "context"}
	}

	if value := strings.TrimSpace(input.SessionDefaultValue); value != "" {
		return Output{Value: value, Source: "session"}
	}

	if value := strings.TrimSpace(input.UserDefaultValue); value != "" {
		return Output{Value: value, Source: "user"}
	}

	return Output{}
}
