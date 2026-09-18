// 本文件定义跨协议生成控制及其取值约束。
package llm

import (
	"fmt"
	"math"
)

// GenerationOptions 保留调用方明确设置的生成参数。
type GenerationOptions struct {
	// MaxOutputTokens 是本次输出上限，nil 表示未指定。
	MaxOutputTokens *int
	// Temperature 是采样温度，零值通过指针保留。
	Temperature *float64
	// TopP 是核采样概率。
	TopP *float64
	// TopK 是候选数量。
	TopK *int
	// ReasoningEffort 是调用方要求的推理档位，由供应商明确映射或拒绝。
	ReasoningEffort string
	// ToolChoice 指定自动、禁止、必须或命名工具选择。
	ToolChoice *ToolSelection
}

// ToolSelection 表示工具选择语义，不携带任何供应商协议结构。
type ToolSelection struct {
	// Mode 为 auto、none、required 或 named。
	Mode string
	// Name 仅在 named 模式指定目标工具。
	Name string
}

// Validate 拒绝不合法的生成参数，避免转成无符号数后溢出。
func (g GenerationOptions) Validate() error {
	if g.MaxOutputTokens != nil && *g.MaxOutputTokens <= 0 {
		return fmt.Errorf("max output tokens must be positive")
	}
	if g.Temperature != nil && (math.IsNaN(*g.Temperature) || math.IsInf(*g.Temperature, 0) || *g.Temperature < 0 || *g.Temperature > 2) {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if g.TopP != nil && (math.IsNaN(*g.TopP) || math.IsInf(*g.TopP, 0) || *g.TopP < 0 || *g.TopP > 1) {
		return fmt.Errorf("top_p must be between 0 and 1")
	}
	if g.TopK != nil && *g.TopK <= 0 {
		return fmt.Errorf("top_k must be positive")
	}
	if c := g.ToolChoice; c != nil {
		switch c.Mode {
		case "auto", "none", "required":
		case "named":
			if c.Name == "" {
				return fmt.Errorf("named tool_choice requires a name")
			}
		default:
			return fmt.Errorf("unsupported tool_choice mode %q", c.Mode)
		}
	}
	return nil
}
