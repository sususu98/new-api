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
	case constant.ChannelTypeSora:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAIVideo}
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

// GetRequiredEndpointTypeByRequestPath returns the client endpoint that must be
// honored during channel selection. Unknown paths are left unfiltered so legacy
// cross-format conversion behavior is preserved.
func GetRequiredEndpointTypeByRequestPath(path string) (constant.EndpointType, bool) {
	if path == "" {
		return "", false
	}
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	path = strings.TrimRight(strings.ToLower(path), "/")
	switch {
	case strings.HasPrefix(path, "/v1/responses/compact"):
		return constant.EndpointTypeOpenAIResponseCompact, true
	case strings.HasPrefix(path, "/v1/responses"):
		return constant.EndpointTypeOpenAIResponse, true
	default:
		return "", false
	}
}

// ChannelSupportsEndpointType checks whether the channel adaptor can accept a
// client request for the required endpoint type.
func ChannelSupportsEndpointType(channelType int, endpointType constant.EndpointType) bool {
	if endpointType == "" {
		return true
	}
	switch endpointType {
	case constant.EndpointTypeOpenAIResponse:
		switch channelType {
		case constant.ChannelTypeOpenAI,
			constant.ChannelTypeAzure,
			constant.ChannelTypeCodex,
			constant.ChannelTypeXai,
			constant.ChannelTypeAli,
			constant.ChannelCloudflare,
			constant.ChannelTypeVolcEngine,
			constant.ChannelTypePerplexity:
			return true
		}
	case constant.EndpointTypeOpenAIResponseCompact:
		switch channelType {
		case constant.ChannelTypeOpenAI,
			constant.ChannelTypeAzure,
			constant.ChannelTypeCodex:
			return true
		}
	}
	return false
}
