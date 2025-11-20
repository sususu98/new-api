package oaichat

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	relaymedia "github.com/QuantumNous/new-api/relaykit/relayconvert/internal/media"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIChatRequestToClaudeMessagesNormalizesToolInputSchema(t *testing.T) {
	tests := []struct {
		name       string
		parameters any
		wantSchema map[string]any
	}{
		{
			name:       "omitted parameters",
			parameters: nil,
			wantSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			name: "missing type and properties",
			parameters: map[string]any{
				"additionalProperties": false,
			},
			wantSchema: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
		},
		{
			name: "non-string type",
			parameters: map[string]any{
				"type":       123,
				"properties": map[string]any{},
			},
			wantSchema: map[string]any{
				"type":       123,
				"properties": map[string]any{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxTokens := uint(1024)
			got, err := OpenAIChatRequestToClaudeMessages(context.Background(), nil, dto.GeneralOpenAIRequest{
				Model:     "claude-test",
				MaxTokens: &maxTokens,
				Messages: []dto.Message{
					{Role: "user", Content: "Call the tool."},
				},
				Tools: []dto.ToolCallRequest{
					{
						Type: "function",
						Function: dto.FunctionRequest{
							Name:        "get_current_time",
							Description: "Get the current time",
							Parameters:  tt.parameters,
						},
					},
				},
			})

			require.NoError(t, err)
			tools, ok := got.Tools.([]any)
			require.True(t, ok)
			require.Len(t, tools, 1)
			tool, ok := tools[0].(*dto.Tool)
			require.True(t, ok)
			assert.Equal(t, "get_current_time", tool.Name)
			assert.Equal(t, tt.wantSchema, tool.InputSchema)
		})
	}
}

func TestOpenAIChatRequestToClaudeMessagesFiles(t *testing.T) {
	const text = "hello 世界\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	tests := []struct {
		name         string
		filename     string
		detectedMIME string
		wantType     string
		wantMIME     string
		invalidData  bool
		missingData  bool
		resolverErr  error
		wantErr      string
	}{
		{name: "text extension overrides detection", filename: "note.TXT", detectedMIME: "application/pdf", wantType: "text"},
		{name: "markdown", filename: "note.md", detectedMIME: "application/octet-stream", wantType: "text"},
		{name: "json is text", filename: "data.json", detectedMIME: "application/json", wantType: "text"},
		{name: "pdf extension overrides detection", filename: "report.PDF", detectedMIME: "text/plain", wantType: "document", wantMIME: "application/pdf"},
		{name: "jpeg extension overrides detection", filename: "photo.JFIF", detectedMIME: "text/plain", wantType: "image", wantMIME: "image/jpeg"},
		{name: "png", filename: "photo.png", detectedMIME: "application/octet-stream", wantType: "image", wantMIME: "image/png"},
		{name: "no extension detected text", filename: "note", detectedMIME: "text/csv", wantType: "text"},
		{name: "no filename detected pdf", detectedMIME: "application/pdf", wantType: "document", wantMIME: "application/pdf"},
		{name: "trailing dot detected image", filename: "photo.", detectedMIME: "image/webp", wantType: "image", wantMIME: "image/webp"},
		{name: "unknown extension does not fall back", filename: "file.unknown", detectedMIME: "text/plain"},
		{name: "unsupported audio extension", filename: "audio.mp3", detectedMIME: "image/png"},
		{name: "unsupported detected type", filename: "file", detectedMIME: "application/zip"},
		{name: "missing file data", filename: "note.txt", missingData: true},
		{name: "invalid text base64", filename: "note.txt", invalidData: true, wantErr: "decode text file failed: illegal base64 data at input byte 0"},
		{name: "resolver failure", filename: "report.pdf", resolverErr: errors.New("unavailable"), wantErr: "get file data failed: unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := false
			relaymedia.SetMediaResolver(relaymedia.MediaResolver{
				GetBase64Data: func(_ context.Context, source types.FileSource, reason ...string) (string, string, error) {
					resolved = true
					assert.Equal(t, encoded, source.GetRawData())
					assert.Equal(t, []string{"formatting file for Claude"}, reason)
					if tt.invalidData {
						return "!invalid", tt.detectedMIME, nil
					}
					return encoded, tt.detectedMIME, tt.resolverErr
				},
			})
			t.Cleanup(func() { relaymedia.SetMediaResolver(relaymedia.MediaResolver{}) })
			fileData := encoded
			if tt.missingData {
				fileData = ""
			}
			// Exercise the client JSON filename field, not only typed content.
			got, err := OpenAIChatRequestToClaudeMessages(context.Background(), nil, dto.GeneralOpenAIRequest{
				Model:     "claude-test",
				MaxTokens: kitutil.GetPointer(uint(1024)),
				Messages: []dto.Message{{Role: "user", Content: []any{
					map[string]any{"type": "text", "text": "Read this:"},
					map[string]any{"type": "file", "file": map[string]any{"filename": tt.filename, "file_data": fileData}},
				}}},
			})
			assert.Equal(t, !tt.missingData, resolved)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Len(t, got.Messages, 1)
			parts, ok := got.Messages[0].Content.([]dto.ClaudeMediaMessage)
			require.True(t, ok)
			want := []dto.ClaudeMediaMessage{{Type: "text", Text: kitutil.GetPointer("Read this:")}}
			switch tt.wantType {
			case "text":
				want = append(want, dto.ClaudeMediaMessage{Type: "text", Text: kitutil.GetPointer(text)})
			case "document", "image":
				want = append(want, dto.ClaudeMediaMessage{Type: tt.wantType, Source: &dto.ClaudeMessageSource{
					Type: "base64", MediaType: tt.wantMIME, Data: encoded,
				}})
			}
			assert.Equal(t, want, parts)
		})
	}
}
