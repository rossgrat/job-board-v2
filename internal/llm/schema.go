package llm

var TriageSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"is_technical": map[string]any{"type": "boolean"},
	},
	"required":             []string{"is_technical"},
	"additionalProperties": false,
}

var NormalizeSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"title": map[string]any{"type": "string"},
		"locations": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"country": map[string]any{"type": "string"},
					"city":    map[string]any{"type": []string{"string", "null"}},
					"setting": map[string]any{"type": "string", "enum": []string{"remote", "hybrid", "onsite"}},
				},
				"required":             []string{"country", "setting"},
				"additionalProperties": false,
			},
		},
		"technologies": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
		"salary_min": map[string]any{"type": []string{"integer", "null"}},
		"salary_max": map[string]any{"type": []string{"integer", "null"}},
		"level":      map[string]any{"type": "string", "enum": []string{"junior", "mid", "senior", "staff", "principal", "management", "unknown"}},
	},
	"required":             []string{"title", "locations", "technologies", "salary_min", "salary_max", "level"},
	"additionalProperties": false,
}

var ClassifySchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"category": map[string]any{
			"type": "string",
			"enum": []string{"backend_engineer", "embedded_firmware", "linux_kernel_networking", "other_interesting", "not_relevant"},
		},
		"relevance": map[string]any{
			"anyOf": []any{
				map[string]any{"type": "string", "enum": []string{"strong_match", "good_match", "partial_match", "weak_match"}},
				map[string]any{"type": "null"},
			},
		},
		"reasoning": map[string]any{"type": "string"},
	},
	"required":             []string{"category", "relevance", "reasoning"},
	"additionalProperties": false,
}
