package service

import (
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEstimateToken_NormalTextReasonable(t *testing.T) {
	en := EstimateToken(OpenAI, "Hello world, this is a short English sentence.")
	assert.Greater(t, en, 5)
	assert.Less(t, en, 30)

	zh := EstimateToken(OpenAI, "这是一段用于估算的中文测试文本。")
	assert.Greater(t, zh, 5)
	assert.Less(t, zh, 40)
}

func TestEstimateToken_Base64NotMultiMillion(t *testing.T) {
	// 1 MiB of base64-like mixed alphanumerics used to thrash letter↔digit
	// transitions and explode into multi-million "tokens".
	var b strings.Builder
	b.Grow(1 << 20)
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	for i := 0; i < 1<<20; i++ {
		b.WriteByte(alphabet[i%len(alphabet)])
	}
	text := b.String()
	got := EstimateToken(OpenAI, text)
	// BPE-like: ~1 token / 4 chars → ~262k, plus small weight. Hard ceiling 2× runes.
	assert.Less(t, got, 400_000, "1MiB base64-like text must stay under 400k estimated tokens, got %d", got)
	assert.Greater(t, got, 100_000, "should still count a substantial amount, got %d", got)
}

func TestEstimateToken_DenseJSONCompressed(t *testing.T) {
	item := `{"id":"a1b2c3","n":12345},`
	text := "[" + strings.Repeat(item, 10_000) + "]"
	got := EstimateToken(OpenAI, text)
	runes := utf8.RuneCountInString(text)
	assert.LessOrEqual(t, got, runes*2)
	assert.Less(t, got, runes, "dense JSON should compress below 1 token/rune, got %d for %d runes", got, runes)
}

func TestEstimateToken_BoundedByRuneCount(t *testing.T) {
	text := strings.Repeat("∑", 1000)
	got := EstimateToken(Claude, text)
	runes := utf8.RuneCountInString(text)
	assert.LessOrEqual(t, got, runes*2)
}

func TestClampLocalBillingPromptTokens(t *testing.T) {
	assert.Equal(t, 0, ClampLocalBillingPromptTokens(-1))
	assert.Equal(t, 100, ClampLocalBillingPromptTokens(100))
	assert.Equal(t, MaxLocalBillingPromptTokens, ClampLocalBillingPromptTokens(MaxLocalBillingPromptTokens+1))
	assert.Equal(t, MaxLocalBillingPromptTokens, ClampLocalBillingPromptTokens(5_557_795))
}

func TestResponseText2Usage_ClampsPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	usage := ResponseText2Usage(c, "hi there", "glm-5.2", 5_557_795)
	require.NotNil(t, usage)
	assert.Equal(t, MaxLocalBillingPromptTokens, usage.PromptTokens)
	assert.Greater(t, usage.CompletionTokens, 0)
	assert.Equal(t, usage.PromptTokens+usage.CompletionTokens, usage.TotalTokens)
}

func TestEstimateTokenByModel_GlmUsesOpenAIWeights(t *testing.T) {
	text := "hello glm model 你好"
	a := EstimateTokenByModel("glm-5.2", text)
	b := EstimateToken(OpenAI, text)
	assert.Equal(t, b, a)
}


func TestCountTextToken_UsesTokenizerForGlm(t *testing.T) {
	// glm has no native codec; must still use cl100k_base, not the heuristic.
	text := "Hello world. This is a tokenizer path check for glm-5.2."
	got := CountTextToken(text, "glm-5.2")
	enc := ensureDefaultTokenEncoder()
	want, err := enc.Count(text)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	// Sanity: far below pathological multi-million estimates
	assert.Less(t, got, 50)
}

func TestCountTextToken_LargeAlnumUsesTokenizer(t *testing.T) {
	text := strings.Repeat("Ab1", 100_000) // 300k chars
	got := CountTextToken(text, "glm-5.2")
	// cl100k on dense alnum is ~chars/something, but must stay well under 5M
	assert.Less(t, got, 500_000, "got %d", got)
	assert.Greater(t, got, 10_000)
}
