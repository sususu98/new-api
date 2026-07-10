package service

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
