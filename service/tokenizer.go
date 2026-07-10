package service

import (
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/tiktoken-go/tokenizer"
	"github.com/tiktoken-go/tokenizer/codec"
)

// tokenEncoderMap won't grow after initialization
var defaultTokenEncoder tokenizer.Codec

// tokenEncoderMap is used to store token encoders for different models
var tokenEncoderMap = make(map[string]tokenizer.Codec)

// tokenEncoderMutex protects tokenEncoderMap for concurrent access
var tokenEncoderMutex sync.RWMutex

func InitTokenEncoders() {
	common.SysLog("initializing token encoders")
	ensureDefaultTokenEncoder()
	common.SysLog("token encoders initialized")
}

func ensureDefaultTokenEncoder() tokenizer.Codec {
	if defaultTokenEncoder != nil {
		return defaultTokenEncoder
	}
	tokenEncoderMutex.Lock()
	defer tokenEncoderMutex.Unlock()
	if defaultTokenEncoder == nil {
		// cl100k_base is a solid general-purpose proxy for non-OpenAI models
		// (glm/claude/gemini/...) when their native tokenizer is unavailable.
		defaultTokenEncoder = codec.NewCl100kBase()
	}
	return defaultTokenEncoder
}

func getTokenEncoder(model string) tokenizer.Codec {
	ensureDefaultTokenEncoder()

	// First, try to get the encoder from cache with read lock
	tokenEncoderMutex.RLock()
	if encoder, exists := tokenEncoderMap[model]; exists {
		tokenEncoderMutex.RUnlock()
		return encoder
	}
	tokenEncoderMutex.RUnlock()

	// If not in cache, create new encoder with write lock
	tokenEncoderMutex.Lock()
	defer tokenEncoderMutex.Unlock()

	// Double-check if another goroutine already created the encoder
	if encoder, exists := tokenEncoderMap[model]; exists {
		return encoder
	}

	// Create new encoder for known OpenAI-family model names when possible.
	modelCodec, err := tokenizer.ForModel(tokenizer.Model(model))
	if err != nil {
		// glm/claude/gemini/etc. have no codec in tiktoken-go; use cl100k_base.
		tokenEncoderMap[model] = defaultTokenEncoder
		return defaultTokenEncoder
	}

	tokenEncoderMap[model] = modelCodec
	return modelCodec
}

func getTokenNum(tokenEncoder tokenizer.Codec, text string) int {
	if text == "" {
		return 0
	}
	if tokenEncoder == nil {
		tokenEncoder = ensureDefaultTokenEncoder()
	}
	tkm, err := tokenEncoder.Count(text)
	if err != nil {
		return EstimateTokenByModel("", text)
	}
	return tkm
}
