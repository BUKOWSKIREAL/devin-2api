// 本文件统一流式模型标识的优先级：上游报告值优先于路由目标和客户端别名。
package common

import "github.com/leookun/devin-2api/internal/llm"

// ResolveStreamModel 更新已知模型标识；晚到的路由目标不能覆盖已确认的上游模型。
func ResolveStreamModel(current string, reported bool, event llm.ResponseEvent) (string, bool) {
	messages := []*llm.AssistantMessage{event.Message, event.Error, event.Partial}
	for _, message := range messages {
		if message != nil && message.ResponseModel != "" {
			return message.ResponseModel, true
		}
	}
	if !reported {
		for _, message := range messages {
			if message != nil && message.Model != "" {
				return message.Model, false
			}
		}
	}
	return current, reported
}
