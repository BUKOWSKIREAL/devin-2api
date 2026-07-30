// 本文件定义 OpenAI Responses 最终 JSON 和 typed SSE 响应编码。
package responses

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/leookun/cha-k/internal/llm"
)

// SSEEvent 是 OpenAI Responses typed SSE 的单个事件。
type SSEEvent struct {
	// Name 是 SSE event 字段值。
	Name string
	// Data 是 JSON 编码的 SSE data 内容。
	Data []byte
}

// EncodeResponse 将最终助手消息编码为非流式 Responses JSON 响应。
func EncodeResponse(message *llm.AssistantMessage) ([]byte, error) {
	if message == nil {
		return nil, fmt.Errorf("response message is nil")
	}
	output := make([]any, 0, len(message.Content))
	for index, block := range message.Content {
		switch content := block.(type) {
		case llm.TextContent:
			output = append(output, map[string]any{
				"type": "message", "id": fmt.Sprintf("msg_%d", index), "role": "assistant",
				"content": []any{map[string]any{"type": "output_text", "text": content.Text, "annotations": []any{}}},
			})
		case llm.ThinkingContent:
			output = append(output, map[string]any{
				"type": "reasoning", "summary": []any{map[string]any{"type": "summary_text", "text": content.Thinking}},
			})
		case llm.ToolCall:
			output = append(output, map[string]any{
				"type": "function_call", "call_id": content.ID, "name": content.Name,
				"arguments": string(content.Arguments), "status": "completed",
			})
		default:
			return nil, fmt.Errorf("unsupported response content type %T", block)
		}
	}
	response := map[string]any{
		"id":         message.ResponseID,
		"object":     "response",
		"created_at": time.UnixMilli(message.TimestampMS).Unix(),
		"model":      message.ResponseModel,
		"status":     responseStatus(message.StopReason),
		"output":     output,
		"usage": map[string]any{
			"input_tokens": message.Usage.Input, "output_tokens": message.Usage.Output, "total_tokens": message.Usage.TotalTokens,
		},
	}
	return json.Marshal(response)
}

// EncodeEvent 将中间响应事件转换为 OpenAI Responses SSE 事件。
func EncodeEvent(event llm.ResponseEvent) (SSEEvent, error) {
	if err := event.Validate(); err != nil {
		return SSEEvent{}, fmt.Errorf("validate response event: %w", err)
	}
	name, payload, err := eventPayload(event)
	if err != nil {
		return SSEEvent{}, err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return SSEEvent{}, fmt.Errorf("encode SSE event %q: %w", name, err)
	}
	return SSEEvent{Name: name, Data: data}, nil
}

func eventPayload(event llm.ResponseEvent) (string, any, error) {
	switch event.Type {
	case llm.ResponseEventStart:
		if event.Partial == nil {
			return "", nil, fmt.Errorf("%s event requires partial message", event.Type)
		}
		return "response.created", map[string]any{
			"type":     "response.created",
			"response": map[string]any{"id": event.Partial.ResponseID, "object": "response"},
		}, nil
	case llm.ResponseEventTextStart:
		return "response.content_part.added", map[string]any{
			"type": "response.content_part.added", "content_index": event.ContentIndex,
		}, nil
	case llm.ResponseEventTextDelta:
		return "response.output_text.delta", map[string]any{
			"type": "response.output_text.delta", "content_index": event.ContentIndex, "delta": event.Delta,
		}, nil
	case llm.ResponseEventTextEnd:
		return "response.output_text.done", map[string]any{
			"type": "response.output_text.done", "content_index": event.ContentIndex, "text": event.Content,
		}, nil
	case llm.ResponseEventThinkingStart:
		return "response.reasoning_summary_part.added", map[string]any{
			"type": "response.reasoning_summary_part.added", "summary_index": event.ContentIndex,
		}, nil
	case llm.ResponseEventThinkingDelta:
		return "response.reasoning_summary_text.delta", map[string]any{
			"type": "response.reasoning_summary_text.delta", "summary_index": event.ContentIndex, "delta": event.Delta,
		}, nil
	case llm.ResponseEventThinkingEnd:
		return "response.reasoning_summary_text.done", map[string]any{
			"type": "response.reasoning_summary_text.done", "summary_index": event.ContentIndex, "text": event.Content,
		}, nil
	case llm.ResponseEventToolCallStart:
		return "response.output_item.added", map[string]any{
			"type": "response.output_item.added", "output_index": event.ContentIndex,
		}, nil
	case llm.ResponseEventToolCallDelta:
		return "response.function_call_arguments.delta", map[string]any{
			"type": "response.function_call_arguments.delta", "output_index": event.ContentIndex, "delta": event.Delta,
		}, nil
	case llm.ResponseEventToolCallEnd:
		if event.ToolCall == nil {
			return "", nil, fmt.Errorf("%s event requires tool call", event.Type)
		}
		return "response.function_call_arguments.done", map[string]any{
			"type": "response.function_call_arguments.done", "output_index": event.ContentIndex,
			"call_id": event.ToolCall.ID, "name": event.ToolCall.Name, "arguments": string(event.ToolCall.Arguments),
		}, nil
	case llm.ResponseEventDone:
		if event.Message == nil {
			return "", nil, fmt.Errorf("%s event requires final message", event.Type)
		}
		return "response.completed", map[string]any{
			"type": "response.completed", "response": responseSummary(event.Message),
		}, nil
	case llm.ResponseEventError:
		if event.Error == nil {
			return "", nil, fmt.Errorf("%s event requires error message", event.Type)
		}
		return "error", map[string]any{
			"type": "error", "error": map[string]any{"message": event.Error.ErrorMessage},
		}, nil
	default:
		return "", nil, fmt.Errorf("unsupported response event type %q", event.Type)
	}
}

func responseSummary(message *llm.AssistantMessage) map[string]any {
	model := message.ResponseModel
	if model == "" {
		model = message.Model
	}
	return map[string]any{
		"id":     message.ResponseID,
		"object": "response",
		"model":  model,
		"status": responseStatus(message.StopReason),
		"usage": map[string]any{
			"input_tokens":  message.Usage.Input,
			"output_tokens": message.Usage.Output,
			"total_tokens":  message.Usage.TotalTokens,
		},
	}
}

func responseStatus(reason llm.StopReason) string {
	if reason == llm.StopReasonError || reason == llm.StopReasonAborted {
		return "failed"
	}
	return "completed"
}
