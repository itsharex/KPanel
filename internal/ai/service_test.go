package ai

import (
	"strings"
	"testing"
)

func TestSessionTitleFromMessage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "  查询 CPU 使用情况  ", want: "查询 CPU 使用情况"},
		{input: "检查服务器。然后修复问题", want: "检查服务器"},
		{input: "第一行\n第二行", want: "第一行 第二行"},
	}
	for _, test := range tests {
		if got := sessionTitleFromMessage(test.input); got != test.want {
			t.Errorf("sessionTitleFromMessage(%q)=%q want=%q", test.input, got, test.want)
		}
	}
	long := sessionTitleFromMessage(strings.Repeat("测", 40))
	if len([]rune(long)) != 37 || !strings.HasSuffix(long, "…") {
		t.Fatalf("long title was not safely truncated: %q", long)
	}
}
