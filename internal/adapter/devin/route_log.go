// 本文件记录最小模型路由信息，不接收正文、工具内容或认证凭证。
package devin

import (
	"encoding/json"
	"log"
)

// writeRouteLog 记录发起上游调用前的路由选择，不代表上游已成功执行。
func writeRouteLog(logger *log.Logger, model, effort, upstream, status string) {
	fields := map[string]string{
		"client_model":     boundedRouteField(model),
		"reasoning_effort": boundedRouteField(effort),
		"upstream_model":   boundedRouteField(upstream),
		"status":           status,
	}
	encoded, _ := json.Marshal(fields)
	logger.Printf("devin_route %s", encoded)
}

// boundedRouteField 限制调用方可控字段长度；JSON 编码负责转义换行。
func boundedRouteField(value string) string {
	if len(value) > 160 {
		return value[:160] + "...[truncated]"
	}
	return value
}
