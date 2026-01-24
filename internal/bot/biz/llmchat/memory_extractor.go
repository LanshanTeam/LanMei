package llmchat

import (
	"LanMei/internal/bot/utils/llog"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	fmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type MemoryExtraction struct {
	Summary string   `json:"summary"`
	Facts   []string `json:"facts"`
}

type MemoryExtractor struct {
	model    fmodel.ToolCallingChatModel
	template *prompt.DefaultChatTemplate
}

func NewMemoryExtractor(model fmodel.ToolCallingChatModel) *MemoryExtractor {
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是群聊记忆提取器，必须调用工具 extract_memory 输出参数，不要输出其他文本。"),
		schema.SystemMessage("聚合以下事件，记录用户行为与性格特征，忽略短期情绪。"),
		schema.SystemMessage("summary 要覆盖本批次核心事件，facts 使用简洁要点并标明用户名或昵称。"),
		schema.UserMessage("群:{group_id}"),
		schema.UserMessage("事件列表:\n{events_text}"),
	)
	return &MemoryExtractor{
		model:    model,
		template: template,
	}
}

func (e *MemoryExtractor) ExtractBatch(ctx context.Context, groupID string, events []MemoryEvent) MemoryExtraction {
	if e == nil || e.model == nil || e.template == nil {
		return MemoryExtraction{}
	}
	if len(events) == 0 {
		return MemoryExtraction{}
	}
	in, err := e.template.Format(ctx, map[string]any{
		"group_id":    groupID,
		"events_text": formatMemoryEvents(events),
	})
	if err != nil {
		llog.Error("format memory extractor prompt error: %v", err)
		return MemoryExtraction{}
	}
	msg, err := e.model.Generate(ctx, in)
	if err != nil {
		llog.Error("generate memory extraction error: %v", err)
		return MemoryExtraction{}
	}
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name != "extract_memory" {
			continue
		}
		var result MemoryExtraction
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &result); err != nil {
			llog.Error("解析记忆提取工具参数失败: %v", err)
			break
		}
		result.Summary = strings.TrimSpace(result.Summary)
		cleanFacts := make([]string, 0, len(result.Facts))
		for _, fact := range result.Facts {
			fact = strings.TrimSpace(fact)
			if fact == "" {
				continue
			}
			cleanFacts = append(cleanFacts, fact)
		}
		result.Facts = cleanFacts
		return result
	}
	return MemoryExtraction{}
}

func formatMemoryEvents(events []MemoryEvent) string {
	lines := make([]string, 0, len(events))
	for i, event := range events {
		line := fmt.Sprintf("[%d] 用户:%s(%s) 消息:%s 蓝妹:%s", i+1, event.Nickname, event.UserID, event.Input, event.Reply)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
