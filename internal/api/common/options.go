// 本文件解析各协议的工具选择，并对尚未实现的控制选项明确报错。
package common

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/leookun/devin-2api/internal/llm"
)

// HasValue 判断 JSON 字段是否存在且不是 null。
func HasValue(raw json.RawMessage) bool {
	return len(bytes.TrimSpace(raw)) > 0 && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

// RejectUnsupported 拒绝可能被误认为已生效的控制字段；普通元数据仍可忽略。
func RejectUnsupported(data []byte, fields ...string) error {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	for _, field := range fields {
		if HasValue(values[field]) {
			return fmt.Errorf("%s is not supported by this gateway", field)
		}
	}
	return nil
}

// DecodeToolChoice 将 Chat、Responses 和 Messages 的选择结构归一化。
func DecodeToolChoice(raw json.RawMessage) (*llm.ToolSelection, error) {
	if !HasValue(raw) {
		return nil, nil
	}
	var mode string
	if json.Unmarshal(raw, &mode) == nil {
		c := &llm.ToolSelection{Mode: mode}
		if err := (llm.GenerationOptions{ToolChoice: c}).Validate(); err != nil {
			return nil, err
		}
		return c, nil
	}
	var value struct {
		// Type 是各协议的选择类型。
		Type string `json:"type"`
		// Name 是 Responses/Anthropic 的目标工具。
		Name string `json:"name"`
		// Function 是 Chat 协议内嵌的目标工具。
		Function struct {
			// Name 是 Chat 命名函数选择的目标名称。
			Name string `json:"name"`
		} `json:"function"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("invalid tool_choice: %w", err)
	}
	if err := RejectUnsupported(raw, "disable_parallel_tool_use"); err != nil {
		return nil, err
	}
	c := &llm.ToolSelection{Mode: value.Type}
	switch value.Type {
	case "function", "tool":
		c.Mode = "named"
		c.Name = value.Name
		if c.Name == "" {
			c.Name = value.Function.Name
		}
	case "any":
		c.Mode = "required"
	}
	if err := (llm.GenerationOptions{ToolChoice: c}).Validate(); err != nil {
		return nil, err
	}
	return c, nil
}
