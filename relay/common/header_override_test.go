package common

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// setupTestContext 创建测试用的 gin.Context
func setupTestContext(headers map[string]string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)

	for key, value := range headers {
		c.Request.Header.Set(key, value)
	}

	return c
}

// TestTryParseHeaderOperations_ValidConfig 测试有效配置的解析
func TestTryParseHeaderOperations_ValidConfig(t *testing.T) {
	config := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"header": "User-Agent",
				"value":  "custom-agent",
			},
		},
	}

	operations, ok := TryParseHeaderOperations(config)
	if !ok {
		t.Fatal("Expected successful parsing")
	}
	if len(operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(operations))
	}
	if operations[0].Header != "User-Agent" {
		t.Errorf("Expected header 'User-Agent', got '%s'", operations[0].Header)
	}
	if operations[0].Value != "custom-agent" {
		t.Errorf("Expected value 'custom-agent', got '%s'", operations[0].Value)
	}
}

// TestTryParseHeaderOperations_MissingHeader 测试缺少 header 字段时解析失败
func TestTryParseHeaderOperations_MissingHeader(t *testing.T) {
	config := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"value": "custom-agent",
			},
		},
	}

	operations, ok := TryParseHeaderOperations(config)
	if ok {
		t.Fatal("Expected parsing to fail when header is missing")
	}
	if operations != nil {
		t.Error("Expected nil operations when parsing fails")
	}
}

// TestTryParseHeaderOperations_EmptyHeader 测试空 header 字段时解析失败
func TestTryParseHeaderOperations_EmptyHeader(t *testing.T) {
	config := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"header": "",
				"value":  "custom-agent",
			},
		},
	}

	operations, ok := TryParseHeaderOperations(config)
	if ok {
		t.Fatal("Expected parsing to fail when header is empty")
	}
	if operations != nil {
		t.Error("Expected nil operations when parsing fails")
	}
}

// TestTryParseHeaderOperations_MissingValue 测试缺少 value 字段时解析失败
func TestTryParseHeaderOperations_MissingValue(t *testing.T) {
	config := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"header": "User-Agent",
			},
		},
	}

	operations, ok := TryParseHeaderOperations(config)
	if ok {
		t.Fatal("Expected parsing to fail when value is missing")
	}
	if operations != nil {
		t.Error("Expected nil operations when parsing fails")
	}
}

// TestTryParseHeaderOperations_EmptyOperations 测试空 operations 数组时解析失败
func TestTryParseHeaderOperations_EmptyOperations(t *testing.T) {
	config := map[string]interface{}{
		"operations": []interface{}{},
	}

	operations, ok := TryParseHeaderOperations(config)
	if ok {
		t.Fatal("Expected parsing to fail when operations array is empty")
	}
	if operations != nil {
		t.Error("Expected nil operations when array is empty")
	}
}

// TestTryParseHeaderOperations_WithConditions 测试带条件的配置
func TestTryParseHeaderOperations_WithConditions(t *testing.T) {
	config := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"header": "User-Agent",
				"value":  "custom-agent",
				"conditions": []interface{}{
					map[string]interface{}{
						"header": "User-Agent",
						"mode":   "contains",
						"value":  "claude-cli",
						"invert": true,
					},
				},
				"logic": "OR",
			},
		},
	}

	operations, ok := TryParseHeaderOperations(config)
	if !ok {
		t.Fatal("Expected successful parsing")
	}
	if len(operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(operations))
	}
	if len(operations[0].Conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(operations[0].Conditions))
	}
	cond := operations[0].Conditions[0]
	if cond.Header != "User-Agent" {
		t.Errorf("Expected condition header 'User-Agent', got '%s'", cond.Header)
	}
	if cond.Mode != "contains" {
		t.Errorf("Expected mode 'contains', got '%s'", cond.Mode)
	}
	if cond.Value != "claude-cli" {
		t.Errorf("Expected value 'claude-cli', got '%s'", cond.Value)
	}
	if !cond.Invert {
		t.Error("Expected invert to be true")
	}
}

// TestTryParseHeaderOperations_SkipEmptyConditions 测试跳过空条件
func TestTryParseHeaderOperations_SkipEmptyConditions(t *testing.T) {
	config := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"header": "User-Agent",
				"value":  "custom-agent",
				"conditions": []interface{}{
					map[string]interface{}{
						"header": "", // 空 header
						"value":  "test",
					},
					map[string]interface{}{
						"header": "User-Agent",
						"value":  "", // 空 value
					},
					map[string]interface{}{
						"header": "User-Agent",
						"value":  "claude-cli", // 有效条件
					},
				},
			},
		},
	}

	operations, ok := TryParseHeaderOperations(config)
	if !ok {
		t.Fatal("Expected successful parsing")
	}
	if len(operations[0].Conditions) != 1 {
		t.Fatalf("Expected 1 valid condition (2 should be skipped), got %d", len(operations[0].Conditions))
	}
	if operations[0].Conditions[0].Value != "claude-cli" {
		t.Error("Expected the valid condition to be preserved")
	}
}

// TestCheckSingleHeaderCondition_Contains 测试 contains 模式
func TestCheckSingleHeaderCondition_Contains(t *testing.T) {
	c := setupTestContext(map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows) AppleWebKit",
	})

	tests := []struct {
		name     string
		cond     HeaderCondition
		expected bool
	}{
		{
			name: "contains match",
			cond: HeaderCondition{
				Header: "User-Agent",
				Mode:   "contains",
				Value:  "Mozilla",
			},
			expected: true,
		},
		{
			name: "contains no match",
			cond: HeaderCondition{
				Header: "User-Agent",
				Mode:   "contains",
				Value:  "claude-cli",
			},
			expected: false,
		},
		{
			name: "contains match with invert",
			cond: HeaderCondition{
				Header: "User-Agent",
				Mode:   "contains",
				Value:  "claude-cli",
				Invert: true,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkSingleHeaderCondition(c, tt.cond)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestCheckSingleHeaderCondition_Modes 测试不同的比较模式
func TestCheckSingleHeaderCondition_Modes(t *testing.T) {
	c := setupTestContext(map[string]string{
		"User-Agent": "Mozilla/5.0",
	})

	tests := []struct {
		name     string
		mode     string
		value    string
		expected bool
	}{
		{"full match", "full", "Mozilla/5.0", true},
		{"full no match", "full", "Mozilla", false},
		{"prefix match", "prefix", "Mozilla", true},
		{"prefix no match", "prefix", "Chrome", false},
		{"suffix match", "suffix", "5.0", true},
		{"suffix no match", "suffix", "6.0", false},
		{"contains match", "contains", "Mozilla", true},
		{"contains no match", "contains", "Chrome", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := HeaderCondition{
				Header: "User-Agent",
				Mode:   tt.mode,
				Value:  tt.value,
			}
			result := checkSingleHeaderCondition(c, cond)
			if result != tt.expected {
				t.Errorf("Mode %s with value %s: expected %v, got %v", tt.mode, tt.value, tt.expected, result)
			}
		})
	}
}

// TestCheckHeaderConditions_AndLogic 测试 AND 逻辑
func TestCheckHeaderConditions_AndLogic(t *testing.T) {
	c := setupTestContext(map[string]string{
		"User-Agent": "Mozilla/5.0",
	})

	conditions := []HeaderCondition{
		{Header: "User-Agent", Mode: "contains", Value: "Mozilla"},
		{Header: "User-Agent", Mode: "contains", Value: "5.0"},
	}

	result := checkHeaderConditions(c, conditions, "AND")
	if !result {
		t.Error("Expected AND logic to pass when all conditions match")
	}

	conditions[1].Value = "Chrome"
	result = checkHeaderConditions(c, conditions, "AND")
	if result {
		t.Error("Expected AND logic to fail when one condition doesn't match")
	}
}

// TestCheckHeaderConditions_OrLogic 测试 OR 逻辑
func TestCheckHeaderConditions_OrLogic(t *testing.T) {
	c := setupTestContext(map[string]string{
		"User-Agent": "Mozilla/5.0",
	})

	conditions := []HeaderCondition{
		{Header: "User-Agent", Mode: "contains", Value: "Chrome"},
		{Header: "User-Agent", Mode: "contains", Value: "Mozilla"},
	}

	result := checkHeaderConditions(c, conditions, "OR")
	if !result {
		t.Error("Expected OR logic to pass when at least one condition matches")
	}

	conditions[1].Value = "Safari"
	result = checkHeaderConditions(c, conditions, "OR")
	if result {
		t.Error("Expected OR logic to fail when no conditions match")
	}
}

// TestApplyHeaderOperations_Basic 测试基本的头覆盖
func TestApplyHeaderOperations_Basic(t *testing.T) {
	c := setupTestContext(map[string]string{
		"User-Agent": "curl/7.68.0",
	})

	info := &RelayInfo{
		ChannelMeta: &ChannelMeta{
			ApiKey: "test-api-key",
		},
	}

	operations := []HeaderOperation{
		{
			Header: "User-Agent",
			Value:  "custom-agent",
		},
	}

	result := ApplyHeaderOperations(c, operations, info)
	if result["User-Agent"] != "custom-agent" {
		t.Errorf("Expected 'custom-agent', got '%s'", result["User-Agent"])
	}
}

// TestApplyHeaderOperations_WithConditions 测试带条件的头覆盖
func TestApplyHeaderOperations_WithConditions(t *testing.T) {
	tests := []struct {
		name           string
		userAgent      string
		expectedResult string
		shouldOverride bool
	}{
		{
			name:           "should override when UA doesn't contain claude-cli",
			userAgent:      "Mozilla/5.0",
			expectedResult: "claude-cli/2.0.37",
			shouldOverride: true,
		},
		{
			name:           "should not override when UA contains claude-cli",
			userAgent:      "claude-cli/2.0.37 (external, cli)",
			expectedResult: "",
			shouldOverride: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := setupTestContext(map[string]string{
				"User-Agent": tt.userAgent,
			})

			info := &RelayInfo{
				ChannelMeta: &ChannelMeta{
					ApiKey: "test-api-key",
				},
			}

			operations := []HeaderOperation{
				{
					Header: "User-Agent",
					Value:  "claude-cli/2.0.37",
					Conditions: []HeaderCondition{
						{
							Header: "User-Agent",
							Mode:   "contains",
							Value:  "claude-cli",
							Invert: true,
						},
					},
				},
			}

			result := ApplyHeaderOperations(c, operations, info)
			if tt.shouldOverride {
				if result["User-Agent"] != tt.expectedResult {
					t.Errorf("Expected '%s', got '%s'", tt.expectedResult, result["User-Agent"])
				}
			} else {
				if _, exists := result["User-Agent"]; exists {
					t.Error("Expected no override, but User-Agent was set")
				}
			}
		})
	}
}

// TestReplaceHeaderVariables 测试变量替换
func TestReplaceHeaderVariables(t *testing.T) {
	info := &RelayInfo{
		ChannelMeta: &ChannelMeta{
			ApiKey: "sk-123456",
		},
	}
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "replace api_key",
			input:    "Bearer {api_key}",
			expected: "Bearer sk-123456",
		},
		{
			name:     "no variable",
			input:    "Bearer token",
			expected: "Bearer token",
		},
		{
			name:     "multiple api_key",
			input:    "{api_key}:{api_key}",
			expected: "sk-123456:sk-123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceHeaderVariables(tt.input, info)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTryParseHeaderOperations_EmptyValue 测试空 value 字段时解析失败
func TestTryParseHeaderOperations_EmptyValue(t *testing.T) {
	config := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"header": "User-Agent",
				"value":  "",
			},
		},
	}

	operations, ok := TryParseHeaderOperations(config)
	if ok {
		t.Fatal("Expected parsing to fail when value is empty")
	}
	if operations != nil {
		t.Error("Expected nil operations when parsing fails")
	}
}

// TestCheckSingleHeaderCondition_EmptyMode 测试空 mode 应该使用默认值 contains
func TestCheckSingleHeaderCondition_EmptyMode(t *testing.T) {
	c := setupTestContext(map[string]string{
		"User-Agent": "Mozilla/5.0",
	})

	// mode 为空字符串应该使用默认的 contains
	condition := HeaderCondition{
		Header: "User-Agent",
		Mode:   "", // 空字符串
		Value:  "Mozilla",
	}

	result := checkSingleHeaderCondition(c, condition)
	// 因为我们在解析时会将空 mode 设置为 "contains"
	// 但这里直接测试 checkSingleHeaderCondition，空 mode 会进入 default 分支返回 false
	if result {
		t.Error("Empty mode should fall into default case and return false")
	}
}

// TestReplaceHeaderVariables_NilInfo 测试 nil info 不会 panic
func TestReplaceHeaderVariables_NilInfo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		info     *RelayInfo
		expected string
	}{
		{
			name:     "nil info",
			input:    "Bearer {api_key}",
			info:     nil,
			expected: "Bearer {api_key}", // 应该原样返回
		},
		{
			name:  "nil ChannelMeta",
			input: "Bearer {api_key}",
			info: &RelayInfo{
				ChannelMeta: nil,
			},
			expected: "Bearer {api_key}", // 应该原样返回
		},
		{
			name:  "valid info",
			input: "Bearer {api_key}",
			info: &RelayInfo{
				ChannelMeta: &ChannelMeta{
					ApiKey: "sk-test",
				},
			},
			expected: "Bearer sk-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceHeaderVariables(tt.input, tt.info)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTryParseHeaderOperations_InvalidConditionsType 测试 conditions 字段类型错误时触发安全回退
func TestTryParseHeaderOperations_InvalidConditionsType(t *testing.T) {
	tests := []struct {
		name       string
		conditions interface{}
	}{
		{
			name:       "conditions is string",
			conditions: "not-an-array",
		},
		{
			name:       "conditions is object",
			conditions: map[string]interface{}{"header": "User-Agent"},
		},
		{
			name:       "conditions is number",
			conditions: 123,
		},
		{
			name:       "conditions is bool",
			conditions: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"header":     "User-Agent",
						"value":      "custom-agent",
						"conditions": tt.conditions,
					},
				},
			}

			operations, ok := TryParseHeaderOperations(config)
			if ok {
				t.Fatal("Expected parsing to fail when conditions field has wrong type")
			}
			if operations != nil {
				t.Error("Expected nil operations when parsing fails")
			}
		})
	}
}

// TestTryParseHeaderOperations_OperationsAsString 测试 operations 以字符串形式提供时也能解析
func TestTryParseHeaderOperations_OperationsAsString(t *testing.T) {
	jsonStr := `[
		{
			"header": "User-Agent",
			"value": "claude-cli/2.0.50 (external, cli)",
			"conditions": [
				{
					"header": "User-Agent",
					"mode": "contains",
					"value": "claude-cli",
					"invert": true
				}
			]
		}
	]`

	config := map[string]interface{}{
		"operations": jsonStr,
	}

	operations, ok := TryParseHeaderOperations(config)
	if !ok {
		t.Fatal("Expected parsing to succeed when operations is a JSON string")
	}
	if len(operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(operations))
	}
	if operations[0].Header != "User-Agent" || operations[0].Value != "claude-cli/2.0.50 (external, cli)" {
		t.Fatalf("Unexpected operation parsed: %+v", operations[0])
	}
	if len(operations[0].Conditions) != 1 || !operations[0].Conditions[0].Invert {
		t.Fatalf("Unexpected conditions parsed: %+v", operations[0].Conditions)
	}
}

// TestTryParseHeaderOperations_EmptyConditionsArray 测试 conditions 数组为空时触发安全回退
func TestTryParseHeaderOperations_EmptyConditionsArray(t *testing.T) {
	config := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"header":     "User-Agent",
				"value":      "custom-agent",
				"conditions": []interface{}{}, // 空数组
			},
		},
	}

	operations, ok := TryParseHeaderOperations(config)
	if ok {
		t.Fatal("Expected parsing to fail when conditions array is empty")
	}
	if operations != nil {
		t.Error("Expected nil operations when parsing fails")
	}
}

// TestTryParseHeaderOperations_AllConditionsInvalid 测试 conditions 数组中全部元素无��时触发安全回退
func TestTryParseHeaderOperations_AllConditionsInvalid(t *testing.T) {
	tests := []struct {
		name       string
		conditions []interface{}
	}{
		{
			name: "all conditions are strings",
			conditions: []interface{}{
				"invalid-condition-1",
				"invalid-condition-2",
			},
		},
		{
			name: "all conditions missing header",
			conditions: []interface{}{
				map[string]interface{}{
					"value": "test",
				},
				map[string]interface{}{
					"value": "test2",
				},
			},
		},
		{
			name: "all conditions with empty header",
			conditions: []interface{}{
				map[string]interface{}{
					"header": "",
					"value":  "test",
				},
			},
		},
		{
			name: "all conditions missing value",
			conditions: []interface{}{
				map[string]interface{}{
					"header": "User-Agent",
				},
			},
		},
		{
			name: "all conditions with empty value",
			conditions: []interface{}{
				map[string]interface{}{
					"header": "User-Agent",
					"value":  "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"header":     "User-Agent",
						"value":      "custom-agent",
						"conditions": tt.conditions,
					},
				},
			}

			operations, ok := TryParseHeaderOperations(config)
			if ok {
				t.Fatalf("Expected parsing to fail when all conditions are invalid: %s", tt.name)
			}
			if operations != nil {
				t.Errorf("Expected nil operations when parsing fails: %s", tt.name)
			}
		})
	}
}

// TestTryParseHeaderOperations_InvalidLogic 测试 logic 字段非法值时触发安全回退
func TestTryParseHeaderOperations_InvalidLogic(t *testing.T) {
	tests := []struct {
		name  string
		logic string
	}{
		{
			name:  "logic is XOR",
			logic: "XOR",
		},
		{
			name:  "logic is NAND",
			logic: "NAND",
		},
		{
			name:  "logic is typo (ORR)",
			logic: "ORR",
		},
		{
			name:  "logic is typo (ANDD)",
			logic: "ANDD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"header": "User-Agent",
						"value":  "custom-agent",
						"logic":  tt.logic,
					},
				},
			}

			operations, ok := TryParseHeaderOperations(config)
			if ok {
				t.Fatalf("Expected parsing to fail when logic is invalid: %s", tt.name)
			}
			if operations != nil {
				t.Errorf("Expected nil operations when parsing fails: %s", tt.name)
			}
		})
	}
}

// TestTryParseHeaderOperations_ValidLogicCaseInsensitive 测试 logic 字段大小写不敏感
func TestTryParseHeaderOperations_ValidLogicCaseInsensitive(t *testing.T) {
	tests := []struct {
		name          string
		logic         string
		expectedLogic string
	}{
		{
			name:          "logic is lowercase and",
			logic:         "and",
			expectedLogic: "AND",
		},
		{
			name:          "logic is uppercase AND",
			logic:         "AND",
			expectedLogic: "AND",
		},
		{
			name:          "logic is mixed case AnD",
			logic:         "AnD",
			expectedLogic: "AND",
		},
		{
			name:          "logic is lowercase or",
			logic:         "or",
			expectedLogic: "OR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"header": "User-Agent",
						"value":  "custom-agent",
						"logic":  tt.logic,
					},
				},
			}

			operations, ok := TryParseHeaderOperations(config)
			if !ok {
				t.Fatalf("Expected parsing to succeed with valid logic: %s", tt.logic)
			}
			if operations[0].Logic != tt.expectedLogic {
				t.Errorf("Expected logic to be normalized to '%s', got '%s'", tt.expectedLogic, operations[0].Logic)
			}
		})
	}
}

// TestTryParseHeaderOperations_AllConditionsInvalidMode 测试全部条件mode非法时触发安全回退
func TestTryParseHeaderOperations_AllConditionsInvalidMode(t *testing.T) {
	config := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"header": "User-Agent",
				"value":  "custom-agent",
				"conditions": []interface{}{
					map[string]interface{}{
						"header": "User-Agent",
						"value":  "test",
						"mode":   "invalid-mode-1",
					},
					map[string]interface{}{
						"header": "User-Agent",
						"value":  "test2",
						"mode":   "invalid-mode-2",
					},
				},
			},
		},
	}

	operations, ok := TryParseHeaderOperations(config)
	if ok {
		t.Fatal("Expected parsing to fail when all conditions have invalid mode")
	}
	if operations != nil {
		t.Error("Expected nil operations when parsing fails")
	}
}

// TestTryParseHeaderOperations_LogicTypeError 测试 logic 字段类型错误时触发安全回退
func TestTryParseHeaderOperations_LogicTypeError(t *testing.T) {
	tests := []struct {
		name  string
		logic interface{}
	}{
		{
			name:  "logic is number",
			logic: 123,
		},
		{
			name:  "logic is bool",
			logic: true,
		},
		{
			name:  "logic is object",
			logic: map[string]interface{}{"invalid": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"header": "User-Agent",
						"value":  "custom-agent",
						"logic":  tt.logic,
					},
				},
			}

			operations, ok := TryParseHeaderOperations(config)
			if ok {
				t.Fatalf("Expected parsing to fail when logic field has wrong type")
			}
			if operations != nil {
				t.Error("Expected nil operations when parsing fails")
			}
		})
	}
}

// TestTryParseHeaderOperations_ModeTypeError 测试 mode 字段类型错误时跳过该条件
func TestTryParseHeaderOperations_ModeTypeError(t *testing.T) {
	config := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"header": "User-Agent",
				"value":  "custom-agent",
				"conditions": []interface{}{
					map[string]interface{}{
						"header": "User-Agent",
						"value":  "test",
						"mode":   123, // 非字符串类型
					},
				},
			},
		},
	}

	operations, ok := TryParseHeaderOperations(config)
	if ok {
		t.Fatal("Expected parsing to fail when all conditions have type error in mode")
	}
	if operations != nil {
		t.Error("Expected nil operations when parsing fails")
	}
}

// TestTryParseHeaderOperations_InvertTypeError 测试 invert 字段类型错误时跳过该条件
func TestTryParseHeaderOperations_InvertTypeError(t *testing.T) {
	config := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"header": "User-Agent",
				"value":  "custom-agent",
				"conditions": []interface{}{
					map[string]interface{}{
						"header": "User-Agent",
						"value":  "test",
						"invert": "true", // 非布尔类型
					},
				},
			},
		},
	}

	operations, ok := TryParseHeaderOperations(config)
	if ok {
		t.Fatal("Expected parsing to fail when all conditions have type error in invert")
	}
	if operations != nil {
		t.Error("Expected nil operations when parsing fails")
	}
}
