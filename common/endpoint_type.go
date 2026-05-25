package common

import (
	"strings"

	"github.com/QuantumNous/new-api/constant"
)

// GetEndpointTypesByChannelType 获取渠道最优先端点类型（所有的渠道都支持 OpenAI 端点）
func GetEndpointTypesByChannelType(channelType int, modelName string) []constant.EndpointType {
	var endpointTypes []constant.EndpointType
	switch channelType {
	case constant.ChannelTypeJina:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeJinaRerank}
	//case constant.ChannelTypeMidjourney, constant.ChannelTypeMidjourneyPlus:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeMidjourney}
	//case constant.ChannelTypeSunoAPI:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeSuno}
	//case constant.ChannelTypeKling:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeKling}
	//case constant.ChannelTypeJimeng:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeJimeng}
	case constant.ChannelTypeAws:
		fallthrough
	case constant.ChannelTypeAnthropic:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeAnthropic, constant.EndpointTypeOpenAI}
	case constant.ChannelTypeVertexAi:
		fallthrough
	case constant.ChannelTypeGemini:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeGemini, constant.EndpointTypeOpenAI}
	case constant.ChannelTypeOpenRouter: // OpenRouter 只支持 OpenAI 端点
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAI}
	case constant.ChannelTypeXai:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAI, constant.EndpointTypeOpenAIResponse}
	case constant.ChannelTypeVLLM, constant.ChannelTypeSGLang:
		endpointTypes = GetAdvancedCustomPreset(channelType).SupportedEndpointTypesForModel(modelName)
	case constant.ChannelTypeSora:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAIVideo}
	case constant.ChannelTypeSub2API, constant.ChannelTypeNewAPI:
		endpointTypes = []constant.EndpointType{
			constant.EndpointTypeOpenAI,
			constant.EndpointTypeOpenAIResponse,
			constant.EndpointTypeOpenAIResponseCompact,
			constant.EndpointTypeAnthropic,
			constant.EndpointTypeGemini,
			constant.EndpointTypeOpenAIAlphaSearch,
		}
	case constant.ChannelTypeCodex:
		endpointTypes = []constant.EndpointType{
			constant.EndpointTypeOpenAIResponse,
			constant.EndpointTypeOpenAIResponseCompact,
			constant.EndpointTypeOpenAIAlphaSearch,
		}
	default:
		if IsOpenAIResponseOnlyModel(modelName) {
			endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAIResponse}
		} else {
			endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAI}
		}
	}
	if IsImageGenerationModel(modelName) {
		// add to first
		endpointTypes = append([]constant.EndpointType{constant.EndpointTypeImageGeneration}, endpointTypes...)
	}
	return endpointTypes
}

// GetRequiredEndpointTypeByRequestPath identifies Responses endpoints without
// restricting unrelated endpoints that may use cross-protocol conversion.
func GetRequiredEndpointTypeByRequestPath(path string) (constant.EndpointType, bool) {
	path, _, _ = strings.Cut(path, "?")
	path = strings.TrimRight(path, "/")
	switch {
	case path == "/v1/responses/compact" || strings.HasPrefix(path, "/v1/responses/compact/"):
		return constant.EndpointTypeOpenAIResponseCompact, true
	case path == "/v1/responses" || strings.HasPrefix(path, "/v1/responses/"):
		return constant.EndpointTypeOpenAIResponse, true
	default:
		return "", false
	}
}

// ChannelSupportsEndpointType reports adaptor capability, not a channel's
// configured routes. Advanced Custom and inference presets must also satisfy
// their per-model request-path constraints.
func ChannelSupportsEndpointType(channelType int, endpointType constant.EndpointType) bool {
	if endpointType == "" {
		return true
	}
	apiType, _ := ChannelType2APIType(channelType)
	switch endpointType {
	case constant.EndpointTypeOpenAIResponse:
		// Keep aligned with GetAdaptor and ConvertOpenAIResponsesRequest,
		// including adaptors that convert Responses to another protocol.
		switch apiType {
		case constant.APITypeOpenAI,
			constant.APITypeAnthropic,
			constant.APITypeGemini,
			constant.APITypeDeepSeek,
			constant.APITypeCodex,
			constant.APITypeXai,
			constant.APITypeAli,
			constant.APITypeCloudflare,
			constant.APITypeVolcEngine,
			constant.APITypePerplexity,
			constant.APITypeOllama,
			constant.APITypeOpenRouter,
			constant.APITypeXinference,
			constant.APITypeZhipuV4,
			constant.APITypeAdvancedCustom,
			constant.APITypeSub2API,
			constant.APITypeNewAPI:
			return true
		}
	case constant.EndpointTypeOpenAIResponseCompact:
		return SupportsResponsesCompact(channelType, apiType)
	}
	return false
}
