package contextconfig

import "testing"

func TestResolveValuePrecedence(t *testing.T) {
	tests := []struct {
		name     string
		input    Inputs
		expected Output
	}{
		{
			name: "flag wins",
			input: Inputs{
				FlagValue:          "from-flag",
				PreferredEnvValue:  "from-env",
				ContextConfigValue: "from-context",
			},
			expected: Output{Value: "from-flag", Source: "flag"},
		},
		{
			name: "preferred env wins over legacy env",
			input: Inputs{
				PreferredEnvValue: "preferred",
				LegacyEnvValue:    "legacy",
			},
			expected: Output{Value: "preferred", Source: "env"},
		},
		{
			name: "legacy env used when preferred empty",
			input: Inputs{
				LegacyEnvValue:     "legacy",
				ContextConfigValue: "context",
			},
			expected: Output{Value: "legacy", Source: "env"},
		},
		{
			name: "context used when no flag or env",
			input: Inputs{
				ContextConfigValue: "context",
				UserDefaultValue:   "user",
			},
			expected: Output{Value: "context", Source: "context"},
		},
		{
			name: "session wins over user",
			input: Inputs{
				SessionDefaultValue: "session",
				UserDefaultValue:    "user",
			},
			expected: Output{Value: "session", Source: "session"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := ResolveValue(testCase.input)
			if result != testCase.expected {
				t.Fatalf("expected %+v, got %+v", testCase.expected, result)
			}
		})
	}
}
