// 本文件确保路由日志只有允许的字段，且客户端文本不能伪造多行日志。
package devin

import (
	"bytes"
	"encoding/json"
	"log"
	"strings"
	"testing"
)

func TestRouteLogRecordsOnlyRoutingFields(t *testing.T) {
	var output bytes.Buffer
	writeRouteLog(log.New(&output, "", 0), "swe-2-max", "", "swe-2-max", "selected")
	raw := strings.TrimPrefix(strings.TrimSpace(output.String()), "devin_route ")
	var fields map[string]string
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 4 || fields["client_model"] != "swe-2-max" || fields["reasoning_effort"] != "" || fields["upstream_model"] != "swe-2-max" || fields["status"] != "selected" {
		t.Fatalf("unexpected routing fields: %v", fields)
	}
}

func TestRouteLogEscapesAndBoundsCallerFields(t *testing.T) {
	var output bytes.Buffer
	writeRouteLog(log.New(&output, "", 0), "bad\nforged line", strings.Repeat("x", 1000), "", "rejected")
	if strings.Count(output.String(), "\n") != 1 || len(output.String()) > 500 {
		t.Fatalf("unsafe log shape: %q", output.String())
	}
}
