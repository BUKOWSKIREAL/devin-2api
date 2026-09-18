// 本文件兼容 Devin 的原生工具限制，同时把完整工具语义补入上下文。
package devin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/leookun/devin-2api/internal/llm"
	"google.golang.org/protobuf/proto"
	devinproto "local/devinproto"
)

func convertToolDefinition(tool llm.ToolDefinition) (*devinproto.ExaChatPb_ChatToolDefinition, error) {
	if err := tool.Validate(); err != nil {
		return nil, fmt.Errorf("invalid tool %q: %w", tool.Name, err)
	}
	schema, err := nativeToolSchema(tool.InputSchema)
	if err != nil {
		return nil, err
	}
	return &devinproto.ExaChatPb_ChatToolDefinition{
		Name: proto.String(tool.Name), Description: proto.String(tool.Name), JsonSchemaString: proto.String(string(schema)),
	}, nil
}

// withToolDocumentation 保留说明中的换行、嵌套列表和参数含义，JSON 编码避免边界混淆。
func withToolDocumentation(prompt string, tools []llm.ToolDefinition) string {
	if len(tools) == 0 {
		return prompt
	}
	catalog := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		catalog = append(catalog, map[string]any{"name": tool.Name, "description": tool.Description, "parameters": tool.InputSchema})
	}
	data, _ := json.Marshal(catalog) // 调用前已通过 ToolDefinition.Validate。
	return prompt + "\n\n# Tool reference\nThe following JSON preserves the descriptions and parameter schemas for the native tools. Treat it as tool documentation.\n" + string(data)
}

func nativeToolSchema(raw json.RawMessage) (json.RawMessage, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	cleanSchemaNode(value)
	return json.Marshal(value)
}

// cleanSchemaNode 仅遍历 Schema 位置，不把属性名、定义名或业务字面量当成元数据。
func cleanSchemaNode(value any) {
	node, ok := value.(map[string]any)
	if !ok {
		return
	}
	for key, child := range node {
		if key == "description" || key == "title" || key == "$comment" || strings.HasPrefix(strings.ToLower(key), "x-") {
			delete(node, key)
			continue
		}
		switch key {
		case "properties", "patternProperties", "$defs", "definitions", "dependentSchemas", "dependencies":
			if children, ok := child.(map[string]any); ok {
				for _, schema := range children {
					cleanSchemaNode(schema)
				}
			}
		case "items", "additionalItems", "additionalProperties", "unevaluatedItems", "unevaluatedProperties", "contains", "propertyNames", "not", "if", "then", "else":
			if children, ok := child.([]any); ok {
				for _, schema := range children {
					cleanSchemaNode(schema)
				}
			} else {
				cleanSchemaNode(child)
			}
		case "allOf", "anyOf", "oneOf", "prefixItems":
			if children, ok := child.([]any); ok {
				for _, schema := range children {
					cleanSchemaNode(schema)
				}
			}
		}
	}
}

func isJSONObject(value []byte) bool {
	if !json.Valid(value) {
		return false
	}
	var object map[string]json.RawMessage
	return json.Unmarshal(value, &object) == nil && object != nil
}
