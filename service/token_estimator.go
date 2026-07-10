package service

import (
	"math"
	"strings"
	"sync"
	"unicode"
)

// Provider 定义模型厂商大类
type Provider string

const (
	OpenAI  Provider = "openai"  // 代表 GPT-3.5, GPT-4, GPT-4o
	Gemini  Provider = "gemini"  // 代表 Gemini 1.0, 1.5 Pro/Flash
	Claude  Provider = "claude"  // 代表 Claude 3, 3.5 Sonnet
	Unknown Provider = "unknown" // 兜底默认
)

// multipliers 定义不同厂商的计费权重
type multipliers struct {
	Word       float64 // 英文单词 / 字母数字块
	Number     float64 // 纯数字块（当前与 Word 共用 BPE 块逻辑，保留字段兼容）
	CJK        float64 // 中日韩字符 (每字)
	Symbol     float64 // 普通标点符号
	MathSymbol float64 // 数学符号 (∑,∫,∂,√等，每个)
	URLDelim   float64 // URL分隔符 (/,:,?,&,=,#,%)
	AtSign     float64 // @符号
	Emoji      float64 // Emoji表情 (每个)
	Newline    float64 // 换行符/制表符 (每个)
	Space      float64 // 空格 (每个)
	BasePad    int     // 基础起步消耗 (Start/End tokens)
}

var (
	multipliersMap = map[Provider]multipliers{
		Gemini: {
			Word: 1.15, Number: 2.8, CJK: 0.68, Symbol: 0.38, MathSymbol: 1.05, URLDelim: 1.2, AtSign: 2.5, Emoji: 1.08, Newline: 1.15, Space: 0.2, BasePad: 0,
		},
		Claude: {
			Word: 1.13, Number: 1.63, CJK: 1.21, Symbol: 0.4, MathSymbol: 4.52, URLDelim: 1.26, AtSign: 2.82, Emoji: 2.6, Newline: 0.89, Space: 0.39, BasePad: 0,
		},
		OpenAI: {
			Word: 1.02, Number: 1.55, CJK: 0.85, Symbol: 0.4, MathSymbol: 2.68, URLDelim: 1.0, AtSign: 2.0, Emoji: 2.12, Newline: 0.5, Space: 0.42, BasePad: 0,
		},
	}
	multipliersLock sync.RWMutex
)

// getMultipliers 根据厂商获取权重配置
func getMultipliers(p Provider) multipliers {
	multipliersLock.RLock()
	defer multipliersLock.RUnlock()

	switch p {
	case Gemini:
		return multipliersMap[Gemini]
	case Claude:
		return multipliersMap[Claude]
	case OpenAI:
		return multipliersMap[OpenAI]
	default:
		return multipliersMap[OpenAI]
	}
}

// charsPerToken is the BPE-like average for continuous alphanumeric / dense symbol runs.
// Real tokenizers sit near 4 chars/token for English, identifiers, base64 and JSON punctuation.
const charsPerToken = 4.0

// maxTokensPerRune is a physical upper bound for heuristic estimates.
// CJK-heavy text is ~1–1.5 tokens/rune; 2× is a safe ceiling against runaway counts.
const maxTokensPerRune = 2.0

// MaxLocalBillingPromptTokens caps heuristic prompt tokens used when settling
// without upstream usage (client_gone / incomplete stream). Upstream-reported
// usage is never clamped by this constant.
const MaxLocalBillingPromptTokens = 256_000

// EstimateToken estimates token count with a lightweight heuristic.
//
// Design goals:
//   - Stay close to real BPE behavior for normal text
//   - Avoid pathological over-count on base64 / letter↔digit thrashing / dense JSON
//   - Bound results by text length so multi-million estimates cannot appear from
//     medium-size agent tool dumps (those feed pre-consume and client_gone settlement)
func EstimateToken(provider Provider, text string) int {
	if text == "" {
		return 0
	}

	m := getMultipliers(provider)
	var count float64
	var alnumRun int
	var symbolRunWeight float64
	var symbolRunLen int
	runeCount := 0

	flushAlnum := func() {
		if alnumRun <= 0 {
			return
		}
		// Continuous alphanumerics share one BPE budget. Do NOT restart on
		// letter↔digit switches — that was the main multi-million over-count path
		// for base64 and minified tool output.
		chunks := math.Ceil(float64(alnumRun) / charsPerToken)
		count += chunks * m.Word
		alnumRun = 0
	}

	flushSymbols := func() {
		if symbolRunLen <= 0 {
			return
		}
		// Dense punctuation (JSON braces/quotes/colons, URL query strings) is not
		// one full token per character. First symbol pays full weight; remainder
		// compresses like BPE (~4 chars/token).
		avg := symbolRunWeight / float64(symbolRunLen)
		count += avg
		if symbolRunLen > 1 {
			count += math.Ceil(float64(symbolRunLen-1)/charsPerToken) * avg
		}
		symbolRunLen = 0
		symbolRunWeight = 0
	}

	pushSymbol := func(weight float64) {
		flushAlnum()
		symbolRunLen++
		symbolRunWeight += weight
	}

	for _, r := range text {
		runeCount++

		if unicode.IsSpace(r) {
			flushAlnum()
			flushSymbols()
			if r == '\n' || r == '\t' {
				count += m.Newline
			} else {
				count += m.Space
			}
			continue
		}

		if isCJK(r) {
			flushAlnum()
			flushSymbols()
			count += m.CJK
			continue
		}

		if isEmoji(r) {
			flushAlnum()
			flushSymbols()
			count += m.Emoji
			continue
		}

		if isLatinOrNumber(r) {
			flushSymbols()
			alnumRun++
			continue
		}

		// Symbols / punctuation
		if isMathSymbol(r) {
			flushAlnum()
			flushSymbols()
			// Math symbols are rare and genuinely expensive in some tokenizers.
			count += m.MathSymbol
			continue
		}
		if r == '@' {
			flushAlnum()
			flushSymbols()
			count += m.AtSign
			continue
		}
		if isURLDelim(r) {
			pushSymbol(m.URLDelim)
			continue
		}
		pushSymbol(m.Symbol)
	}

	flushAlnum()
	flushSymbols()

	result := int(math.Ceil(count)) + m.BasePad
	return clampHeuristicTokenCount(result, runeCount)
}

// clampHeuristicTokenCount bounds a heuristic estimate by text length.
func clampHeuristicTokenCount(count int, runeCount int) int {
	if count < 0 {
		return 0
	}
	if runeCount <= 0 {
		return count
	}
	maxByRunes := int(math.Ceil(float64(runeCount) * maxTokensPerRune))
	if maxByRunes < runeCount {
		maxByRunes = runeCount
	}
	if count > maxByRunes {
		return maxByRunes
	}
	return count
}

// ClampLocalBillingPromptTokens bounds local-only prompt estimates for settlement.
func ClampLocalBillingPromptTokens(estimated int) int {
	if estimated < 0 {
		return 0
	}
	if estimated > MaxLocalBillingPromptTokens {
		return MaxLocalBillingPromptTokens
	}
	return estimated
}

// 辅助：判断是否为 CJK 字符
func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		(r >= 0x3040 && r <= 0x30FF) || // 日文
		(r >= 0xAC00 && r <= 0xD7A3) // 韩文
}

// 辅助：判断是否为单词主体 (字母或数字)
func isLatinOrNumber(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

// 辅助：判断是否为Emoji字符
func isEmoji(r rune) bool {
	return (r >= 0x1F300 && r <= 0x1F9FF) ||
		(r >= 0x2600 && r <= 0x26FF) ||
		(r >= 0x2700 && r <= 0x27BF) ||
		(r >= 0x1F600 && r <= 0x1F64F) ||
		(r >= 0x1F900 && r <= 0x1F9FF) ||
		(r >= 0x1FA00 && r <= 0x1FAFF)
}

// 辅助：判断是否为数学符号
func isMathSymbol(r rune) bool {
	mathSymbols := "∑∫∂√∞≤≥≠≈±×÷∈∉∋∌⊂⊃⊆⊇∪∩∧∨¬∀∃∄∅∆∇∝∟∠∡∢°′″‴⁺⁻⁼⁽⁾ⁿ₀₁₂₃₄₅₆₇₈₉₊₋₌₍₎²³¹⁴⁵⁶⁷⁸⁹⁰"
	for _, m := range mathSymbols {
		if r == m {
			return true
		}
	}
	if r >= 0x2200 && r <= 0x22FF {
		return true
	}
	if r >= 0x2A00 && r <= 0x2AFF {
		return true
	}
	if r >= 0x1D400 && r <= 0x1D7FF {
		return true
	}
	return false
}

// 辅助：判断是否为URL分隔符
func isURLDelim(r rune) bool {
	switch r {
	case '/', ':', '?', '&', '=', '#', '%':
		return true
	default:
		return false
	}
}

func EstimateTokenByModel(model, text string) int {
	if text == "" {
		return 0
	}

	model = strings.ToLower(model)
	if strings.Contains(model, "gemini") {
		return EstimateToken(Gemini, text)
	}
	if strings.Contains(model, "claude") {
		return EstimateToken(Claude, text)
	}
	// glm / deepseek / qwen / etc. share the OpenAI-weight heuristic
	return EstimateToken(OpenAI, text)
}
