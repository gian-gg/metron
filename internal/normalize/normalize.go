package normalize

import (
	"math"
	"strconv"
)

// Span kinds.
const (
	KindModel = "model"
	KindTool  = "tool"
	KindOther = "other"
)

// GenAI semantic convention attribute names.
const (
	attrOperation        = "gen_ai.operation.name"
	attrToolName         = "gen_ai.tool.name"
	attrRequestModel     = "gen_ai.request.model"
	attrResponseModel    = "gen_ai.response.model"
	attrInputTokens      = "gen_ai.usage.input_tokens"
	attrOutputTokens     = "gen_ai.usage.output_tokens"
	attrPromptTokens     = "gen_ai.usage.prompt_tokens"
	attrCompletionTokens = "gen_ai.usage.completion_tokens"
)

const (
	opExecuteTool = "execute_tool"
	opInvokeAgent = "invoke_agent"
	opCreateAgent = "create_agent"
)

var modelOperations = map[string]bool{
	"chat":             true,
	"text_completion":  true,
	"generate_content": true,
	"embeddings":       true,
}

// Fields are the span columns derived from its attributes.
type Fields struct {
	Kind         string
	Model        *string
	InputTokens  *int64
	OutputTokens *int64
}

// Span derives Fields from a span's attributes.
func Span(attrs map[string]any) Fields {
	model := firstString(attrs, attrRequestModel, attrResponseModel)
	return Fields{
		Kind:         kind(attrs, model),
		Model:        model,
		InputTokens:  firstCount(attrs, attrInputTokens, attrPromptTokens),
		OutputTokens: firstCount(attrs, attrOutputTokens, attrCompletionTokens),
	}
}

func kind(attrs map[string]any, model *string) string {
	op, _ := attrs[attrOperation].(string)
	switch {
	case op == opExecuteTool || firstString(attrs, attrToolName) != nil:
		return KindTool
	case op == opInvokeAgent || op == opCreateAgent:
		return KindOther
	case model != nil || modelOperations[op]:
		return KindModel
	default:
		return KindOther
	}
}

func firstString(attrs map[string]any, keys ...string) *string {
	for _, k := range keys {
		if s, ok := attrs[k].(string); ok && s != "" {
			return &s
		}
	}
	return nil
}

func firstCount(attrs map[string]any, keys ...string) *int64 {
	for _, k := range keys {
		if n, ok := count(attrs[k]); ok {
			return &n
		}
	}
	return nil
}

// count reads a non-negative whole number from an int, float or numeric string.
func count(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, n >= 0
	case int:
		return int64(n), n >= 0
	case float64:
		if n < 0 || n != math.Trunc(n) || n >= math.MaxInt64 {
			return 0, false
		}
		return int64(n), true
	case string:
		parsed, err := strconv.ParseInt(n, 10, 64)
		return parsed, err == nil && parsed >= 0
	default:
		return 0, false
	}
}
