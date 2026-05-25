package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestGetRandomSatisfiedChannelForEndpointFiltersIncompatibleChannels(t *testing.T) {
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	oldGroup2Model2Channels := group2model2channels
	oldChannelsIDM := channelsIDM
	defer func() {
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
		channelSyncLock.Lock()
		group2model2channels = oldGroup2Model2Channels
		channelsIDM = oldChannelsIDM
		channelSyncLock.Unlock()
	}()

	common.MemoryCacheEnabled = true
	channelSyncLock.Lock()
	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-5.5": {50, 51},
		},
	}
	channelsIDM = map[int]*Channel{
		50: testEndpointChannel(50, constant.ChannelTypeAnthropic, 10, 100),
		51: testEndpointChannel(51, constant.ChannelTypeOpenAI, 10, 100),
	}
	channelSyncLock.Unlock()

	channel, err := GetRandomSatisfiedChannelForEndpoint("default", "gpt-5.5", 0, constant.EndpointTypeOpenAIResponse)
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 51, channel.Id)
}

func TestGetRandomSatisfiedChannelForEndpointReturnsNilWhenOnlyIncompatible(t *testing.T) {
	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	oldGroup2Model2Channels := group2model2channels
	oldChannelsIDM := channelsIDM
	defer func() {
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
		channelSyncLock.Lock()
		group2model2channels = oldGroup2Model2Channels
		channelsIDM = oldChannelsIDM
		channelSyncLock.Unlock()
	}()

	common.MemoryCacheEnabled = true
	channelSyncLock.Lock()
	group2model2channels = map[string]map[string][]int{
		"default": {
			"gpt-5.5": {50},
		},
	}
	channelsIDM = map[int]*Channel{
		50: testEndpointChannel(50, constant.ChannelTypeAnthropic, 10, 100),
	}
	channelSyncLock.Unlock()

	channel, err := GetRandomSatisfiedChannelForEndpoint("default", "gpt-5.5", 0, constant.EndpointTypeOpenAIResponse)
	require.NoError(t, err)
	require.Nil(t, channel)
}

func TestGetChannelForEndpointDBFiltersIncompatibleChannels(t *testing.T) {
	group := "endpoint-db-filter"
	modelName := "o3-pro"
	priority := int64(10)
	weight := uint(100)

	oldMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		common.MemoryCacheEnabled = oldMemoryCacheEnabled
		DB.Where(commonGroupCol+" = ?", group).Delete(&Ability{})
		DB.Where("name LIKE ?", "endpoint-db-%").Delete(&Channel{})
	})

	deepSeekChannel := &Channel{
		Id:       150,
		Type:     constant.ChannelTypeDeepSeek,
		Key:      "deepseek-key",
		Status:   common.ChannelStatusEnabled,
		Name:     "endpoint-db-deepseek",
		Models:   modelName,
		Group:    group,
		Priority: &priority,
		Weight:   &weight,
	}
	openAIChannel := &Channel{
		Id:       151,
		Type:     constant.ChannelTypeOpenAI,
		Key:      "openai-key",
		Status:   common.ChannelStatusEnabled,
		Name:     "endpoint-db-openai",
		Models:   modelName,
		Group:    group,
		Priority: &priority,
		Weight:   &weight,
	}
	require.NoError(t, DB.Create(deepSeekChannel).Error)
	require.NoError(t, DB.Create(openAIChannel).Error)
	require.NoError(t, DB.Create(&[]Ability{
		{
			Group:     group,
			Model:     modelName,
			ChannelId: deepSeekChannel.Id,
			Enabled:   true,
			Priority:  &priority,
			Weight:    weight,
		},
		{
			Group:     group,
			Model:     modelName,
			ChannelId: openAIChannel.Id,
			Enabled:   true,
			Priority:  &priority,
			Weight:    weight,
		},
	}).Error)

	channel, err := GetChannelForEndpoint(group, modelName, 0, constant.EndpointTypeOpenAIResponse)
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, openAIChannel.Id, channel.Id)
}

func testEndpointChannel(id int, channelType int, priority int64, weight uint) *Channel {
	return &Channel{
		Id:       id,
		Type:     channelType,
		Priority: &priority,
		Weight:   &weight,
	}
}
