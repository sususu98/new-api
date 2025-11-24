package common

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
)

// HeaderCondition 请求头条件判断
type HeaderCondition struct {
	Header string `json:"header"` // 要检查的请求头名称
	Mode   string `json:"mode"`   // full, prefix, suffix, contains
	Value  string `json:"value"`  // 匹配的值
	Invert bool   `json:"invert"` // 是否取反
}

// HeaderOperation 请求头覆盖操作
type HeaderOperation struct {
	Header     string            `json:"header"`     // 要覆盖的请求头名称
	Value      string            `json:"value"`      // 覆盖后的值
	Conditions []HeaderCondition `json:"conditions"` // 条件列表
	Logic      string            `json:"logic"`      // AND, OR (默认OR)
}

// checkHeaderConditions 检查请求头条件列表是否满足
func checkHeaderConditions(c *gin.Context, conditions []HeaderCondition, logic string) bool {
	if len(conditions) == 0 {
		return true // 没有条件，直接通过
	}

	results := make([]bool, len(conditions))
	for i, condition := range conditions {
		results[i] = checkSingleHeaderCondition(c, condition)
	}

	if strings.ToUpper(logic) == "AND" {
		// AND逻辑：所有条件都必须满足
		for _, result := range results {
			if !result {
				return false
			}
		}
		return true
	} else {
		// OR逻辑：任意条件满足即可（默认）
		for _, result := range results {
			if result {
				return true
			}
		}
		return false
	}
}

// checkSingleHeaderCondition 检查单个请求头条件
func checkSingleHeaderCondition(c *gin.Context, condition HeaderCondition) bool {
	headerValue := c.Request.Header.Get(condition.Header)

	var result bool
	switch strings.ToLower(condition.Mode) {
	case "full":
		result = headerValue == condition.Value
	case "prefix":
		result = strings.HasPrefix(headerValue, condition.Value)
	case "suffix":
		result = strings.HasSuffix(headerValue, condition.Value)
	case "contains":
		result = strings.Contains(headerValue, condition.Value)
	default:
		result = false
	}

	if condition.Invert {
		result = !result
	}
	return result
}

// isValidLogic 校验 logic 字段是否为有效值
func isValidLogic(logic string) bool {
	upper := strings.ToUpper(logic)
	return upper == "AND" || upper == "OR"
}

// isValidMode 校验 mode 字段是否为有效值
func isValidMode(mode string) bool {
	lower := strings.ToLower(mode)
	return lower == "full" || lower == "prefix" || lower == "suffix" || lower == "contains"
}

// normalizeOperations 尝试将 operations 字段归一化为 []interface{}
func normalizeOperations(opsValue interface{}) ([]interface{}, bool) {
	switch v := opsValue.(type) {
	case []interface{}:
		return v, true
	case []map[string]interface{}:
		// 将 []map 转成 []interface{}，避免类型断言失败
		res := make([]interface{}, len(v))
		for i, item := range v {
			res[i] = item
		}
		return res, true
	case []map[string]string:
		res := make([]interface{}, len(v))
		for i, item := range v {
			// 需要转换成 map[string]interface{}
			newMap := make(map[string]interface{}, len(item))
			for k, val := range item {
				newMap[k] = val
			}
			res[i] = newMap
		}
		return res, true
	case string:
		// operations 被错误地保存为字符串时，尝试解析
		var arr []interface{}
		if err := json.Unmarshal([]byte(v), &arr); err == nil {
			return arr, true
		}
	}
	return nil, false
}

// TryParseHeaderOperations 尝试解析请求头操作配置
func TryParseHeaderOperations(headerOverride map[string]interface{}) ([]HeaderOperation, bool) {
	if headerOverride == nil {
		return nil, false
	}
	if opsValue, exists := headerOverride["operations"]; exists {
		if opsSlice, ok := normalizeOperations(opsValue); ok {
			var operations []HeaderOperation
			for _, op := range opsSlice {
				if opMap, ok := op.(map[string]interface{}); ok {
					operation := HeaderOperation{}

					// 解析必需字段 - header 和 value 是必需的
					if header, ok := opMap["header"].(string); ok && header != "" {
						operation.Header = header
					} else {
						return nil, false // 缺少必需字段，解析失败
					}
					if value, ok := opMap["value"].(string); ok && value != "" {
						operation.Value = value
					} else {
						return nil, false // 缺少必需字段，解析失败
					}

					// 解析可选字段 - logic 需要类型和白名单校验
					if logicValue, exists := opMap["logic"]; exists {
						logic, ok := logicValue.(string)
						if !ok {
							// logic 字段存在但类型不是字符串，配置错误，触发安全回退
							return nil, false
						}
						if !isValidLogic(logic) {
							// logic 字段值非法，配置错误，触发安全回退
							return nil, false
						}
						operation.Logic = strings.ToUpper(logic)
					} else {
						operation.Logic = "OR" // 默认为OR
					}

					// 解析条件列表
					if conditions, exists := opMap["conditions"]; exists {
						condSlice, ok := conditions.([]interface{})
						if !ok {
							// conditions 字段存在但类型不是数组，配置错误，触发安全回退
							return nil, false
						}
						if len(condSlice) == 0 {
							// conditions 数组为空，配置错误，触发安全回退
							return nil, false
						}
						for _, cond := range condSlice {
							if condMap, ok := cond.(map[string]interface{}); ok {
								condition := HeaderCondition{}
								// 条件中的 header 和 value 都是必需的，空值会导致意外行为
								if header, ok := condMap["header"].(string); ok && header != "" {
									condition.Header = header
								} else {
									continue // header 为空，跳过此条件
								}

								// mode 字段需要类型和白名单校验
								if modeValue, exists := condMap["mode"]; exists {
									mode, ok := modeValue.(string)
									if !ok {
										// mode 字段存在但类型不是字符串，跳过此条件
										continue
									}
									if mode == "" {
										condition.Mode = "contains" // 空字符串使用默认值
									} else {
										if !isValidMode(mode) {
											// mode 字段值非法，跳过此条件
											continue
										}
										condition.Mode = strings.ToLower(mode)
									}
								} else {
									condition.Mode = "contains" // 默认为contains
								}

								if value, ok := condMap["value"].(string); ok && value != "" {
									condition.Value = value
								} else {
									continue // value 为空，跳过此条件
								}

								// invert 字段需要类型校验
								if invertValue, exists := condMap["invert"]; exists {
									invert, ok := invertValue.(bool)
									if !ok {
										// invert 字段存在但类型不是布尔，跳过此条件
										continue
									}
									condition.Invert = invert
								} // 不存在时默认为 false，无需显式赋值

								operation.Conditions = append(operation.Conditions, condition)
							}
						}
						// conditions 数组存在但没有解析出任何有效条件，配置错误，触发安全回退
						if len(operation.Conditions) == 0 {
							return nil, false
						}
					}

					operations = append(operations, operation)
				}
			}
			// 如果解析后 operations 为空，说明配置有误或全部被跳过
			// 返回 false，由调用方决定如何处理（不做头覆盖）
			if len(operations) > 0 {
				return operations, true
			}
		}
	}

	return nil, false
}

// ApplyHeaderOperations 应用请求头操作
func ApplyHeaderOperations(c *gin.Context, operations []HeaderOperation, info *RelayInfo) map[string]string {
	result := make(map[string]string)

	for _, op := range operations {
		// 检查条件是否满足
		if !checkHeaderConditions(c, op.Conditions, op.Logic) {
			// 条件不满足时，透传客户端原始请求头（避免 Go HTTP 客户端使用默认值）
			if originalValue := c.Request.Header.Get(op.Header); originalValue != "" {
				result[op.Header] = originalValue
			}
			continue
		}

		// 应用覆盖，支持变量替换
		value := replaceHeaderVariables(op.Value, info)
		result[op.Header] = value
	}

	return result
}

// replaceHeaderVariables 替换请求头中的变量
func replaceHeaderVariables(str string, info *RelayInfo) string {
	if info == nil || info.ChannelMeta == nil {
		return str
	}
	// 替换 {api_key}
	if strings.Contains(str, "{api_key}") {
		str = strings.ReplaceAll(str, "{api_key}", info.ApiKey)
	}
	// 可扩展更多变量，如 {model}, {user_id} 等
	return str
}
