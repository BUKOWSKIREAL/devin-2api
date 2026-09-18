// 本文件防止工具说明、嵌套参数语义和引用定义在转换时丢失。
package devin

import (
	"encoding/json"
	"github.com/leookun/devin-2api/internal/llm"
	"reflect"
	"strings"
	"testing"
)

func TestToolDefinitionPreservesAnnotationsAndReferences(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"timeout":{"type":"integer","description":"Milliseconds; 0 means unlimited"},"description":{"$ref":"#/$defs/description"}},"$defs":{"description":{"type":"string","title":"business definition"}},"x-custom":{"description":"keep"},"required":["timeout"]}`)
	description := "Usage:\n- Keep order.\n  - Nested rule.\n```sh\na && b\n```"
	got, err := convertToolDefinition(llm.ToolDefinition{Name: "probe", Description: description, InputSchema: schema})
	if err != nil {
		t.Fatal(err)
	}
	if got.GetDescription() != "probe" {
		t.Fatal("native compatibility description changed")
	}
	var native map[string]any
	if err := json.Unmarshal([]byte(got.GetJsonSchemaString()), &native); err != nil {
		t.Fatal(err)
	}
	if _, ok := native["$defs"].(map[string]any)["description"]; !ok {
		t.Fatal("definition name removed")
	}
	prompt := withToolDocumentation("system", []llm.ToolDefinition{{Name: "probe", Description: description, InputSchema: schema}})
	start := strings.Index(prompt, "[{\"")
	if start < 0 {
		t.Fatal("missing documentation")
	}
	var docs []struct {
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	}
	if err := json.Unmarshal([]byte(prompt[start:]), &docs); err != nil {
		t.Fatal(err)
	}
	if docs[0].Description != description {
		t.Fatal("description rewritten")
	}
	var want, actual any
	_ = json.Unmarshal(schema, &want)
	_ = json.Unmarshal(docs[0].Parameters, &actual)
	if !reflect.DeepEqual(want, actual) {
		t.Fatal("schema documentation lost")
	}
}

func TestNativeSchemaKeepsLiteralValuesAndLargeIntegers(t *testing.T) {
	raw := json.RawMessage(`{"type":"object","description":"annotation","properties":{"x-business":{"type":"object","default":{"description":"literal","x-value":9007199254740993},"properties":{"description":{"type":"string","description":"annotation"}}}},"patternProperties":{"^description$":{"type":"string","title":"annotation"}},"$defs":{"x-definition":{"type":"string","description":"annotation"}},"allOf":[{"description":"annotation","type":"object"}]}`)
	got, err := nativeToolSchema(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "9007199254740993") || !strings.Contains(string(got), `"description":"literal"`) || !strings.Contains(string(got), `"x-business"`) || !strings.Contains(string(got), `"x-definition"`) {
		t.Fatalf("business values altered: %s", got)
	}
	if strings.Contains(string(got), `"annotation"`) {
		t.Fatalf("native annotations remain: %s", got)
	}
}
