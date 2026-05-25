package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestGetRequiredEndpointTypeByRequestPath(t *testing.T) {
	for _, tc := range []struct {
		path string
		want constant.EndpointType
	}{
		{"/v1/responses", constant.EndpointTypeOpenAIResponse},
		{"/v1/responses/?foo=bar", constant.EndpointTypeOpenAIResponse},
		{"/v1/responses/resp_123", constant.EndpointTypeOpenAIResponse},
		{"/v1/responses/compact?foo=bar", constant.EndpointTypeOpenAIResponseCompact},
		{"/v1/responses/compact/", constant.EndpointTypeOpenAIResponseCompact},
		{"/v1/responses/compact/nested", constant.EndpointTypeOpenAIResponseCompact},
		{"/v1/responses/compactXYZ", constant.EndpointTypeOpenAIResponse},
		{"/v1/responsesXYZ", ""},
		{"/v1/Responses", ""},
		{"/v1/chat/completions", ""},
		{"/v1/messages", ""},
		{"", ""},
	} {
		t.Run(tc.path, func(t *testing.T) {
			endpoint, required := GetRequiredEndpointTypeByRequestPath(tc.path)
			assert.Equal(t, tc.want, endpoint)
			assert.Equal(t, tc.want != "", required)
		})
	}
}

func TestChannelSupportsEndpointType(t *testing.T) {
	for _, tc := range []struct {
		name               string
		channelType        int
		responses, compact bool
	}{
		{"OpenAI", constant.ChannelTypeOpenAI, true, true},
		{"Azure", constant.ChannelTypeAzure, true, true},
		{"Codex", constant.ChannelTypeCodex, true, true},
		{"NewAPI", constant.ChannelTypeNewAPI, true, true},
		{"Sub2API", constant.ChannelTypeSub2API, true, true},
		{"AdvancedCustom", constant.ChannelTypeAdvancedCustom, true, true},
		{"vLLM", constant.ChannelTypeVLLM, true, true},
		{"SGLang", constant.ChannelTypeSGLang, true, true},
		{"Claude", constant.ChannelTypeAnthropic, true, false},
		{"Gemini", constant.ChannelTypeGemini, true, false},
		{"DeepSeek", constant.ChannelTypeDeepSeek, true, false},
		{"Ollama", constant.ChannelTypeOllama, true, false},
		{"OpenRouter", constant.ChannelTypeOpenRouter, true, false},
		{"Xinference", constant.ChannelTypeXinference, true, false},
		{"ZhipuV4", constant.ChannelTypeZhipu_v4, true, false},
		{"Xai", constant.ChannelTypeXai, true, false},
		{"Ali", constant.ChannelTypeAli, true, false},
		{"Cloudflare", constant.ChannelCloudflare, true, false},
		{"VolcEngine", constant.ChannelTypeVolcEngine, true, false},
		{"Perplexity", constant.ChannelTypePerplexity, true, false},
		{"AWS", constant.ChannelTypeAws, false, false},
		{"Vertex", constant.ChannelTypeVertexAi, false, false},
		{"Moonshot", constant.ChannelTypeMoonshot, false, false},
		{"Zhipu", constant.ChannelTypeZhipu, false, false},
		{"TaskPlugin", constant.ChannelTypeTaskPlugin, false, false},
		{"MiniMax", constant.ChannelTypeMiniMax, false, false},
		{"SiliconFlow", constant.ChannelTypeSiliconFlow, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.responses, ChannelSupportsEndpointType(tc.channelType, constant.EndpointTypeOpenAIResponse))
			assert.Equal(t, tc.compact, ChannelSupportsEndpointType(tc.channelType, constant.EndpointTypeOpenAIResponseCompact))
			assert.True(t, ChannelSupportsEndpointType(tc.channelType, ""))
		})
	}
}
