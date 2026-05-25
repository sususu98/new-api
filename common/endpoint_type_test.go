package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestGetRequiredEndpointTypeByRequestPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want constant.EndpointType
		ok   bool
	}{
		{
			name: "responses",
			path: "/v1/responses",
			want: constant.EndpointTypeOpenAIResponse,
			ok:   true,
		},
		{
			name: "responses compact",
			path: "/v1/responses/compact?foo=bar",
			want: constant.EndpointTypeOpenAIResponseCompact,
			ok:   true,
		},
		{
			name: "chat completions is intentionally unfiltered",
			path: "/v1/chat/completions",
			want: "",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GetRequiredEndpointTypeByRequestPath(tt.path)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestChannelSupportsEndpointTypeResponses(t *testing.T) {
	require.True(t, ChannelSupportsEndpointType(constant.ChannelTypeOpenAI, constant.EndpointTypeOpenAIResponse))
	require.True(t, ChannelSupportsEndpointType(constant.ChannelTypeAzure, constant.EndpointTypeOpenAIResponse))
	require.True(t, ChannelSupportsEndpointType(constant.ChannelTypeCodex, constant.EndpointTypeOpenAIResponse))
	require.True(t, ChannelSupportsEndpointType(constant.ChannelTypeXai, constant.EndpointTypeOpenAIResponse))

	require.False(t, ChannelSupportsEndpointType(constant.ChannelTypeAnthropic, constant.EndpointTypeOpenAIResponse))
	require.False(t, ChannelSupportsEndpointType(constant.ChannelTypeAws, constant.EndpointTypeOpenAIResponse))
	require.False(t, ChannelSupportsEndpointType(constant.ChannelTypeDeepSeek, constant.EndpointTypeOpenAIResponse))
	require.False(t, ChannelSupportsEndpointType(constant.ChannelTypeMoonshot, constant.EndpointTypeOpenAIResponse))
}

func TestChannelSupportsEndpointTypeResponsesCompact(t *testing.T) {
	require.True(t, ChannelSupportsEndpointType(constant.ChannelTypeOpenAI, constant.EndpointTypeOpenAIResponseCompact))
	require.True(t, ChannelSupportsEndpointType(constant.ChannelTypeAzure, constant.EndpointTypeOpenAIResponseCompact))
	require.True(t, ChannelSupportsEndpointType(constant.ChannelTypeCodex, constant.EndpointTypeOpenAIResponseCompact))

	require.False(t, ChannelSupportsEndpointType(constant.ChannelTypeXai, constant.EndpointTypeOpenAIResponseCompact))
	require.False(t, ChannelSupportsEndpointType(constant.ChannelTypeAnthropic, constant.EndpointTypeOpenAIResponseCompact))
}
