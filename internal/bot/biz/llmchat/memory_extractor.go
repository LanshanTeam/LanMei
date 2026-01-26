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
	Sufficient   bool     `json:"sufficient"`
	Participants []string `json:"participants"`
	Cause        string   `json:"cause"`
	Process      string   `json:"process"`
	Result       string   `json:"result"`
}

type MemoryExtractor struct {
	model    fmodel.ToolCallingChatModel
	template *prompt.DefaultChatTemplate
}

func NewMemoryExtractor(model fmodel.ToolCallingChatModel) *MemoryExtractor {
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是群聊记忆整理器，必须调用工具 extract_memory_event 输出参数，不要输出其他文本。"),
		schema.SystemMessage("任务：把下面的聊天记录合并成一条记忆事件，提取主要参与者、起因、经过、结果。"),
		schema.SystemMessage("当 force=false 且信息不足以构成事件（碎片/闲聊/缺少因果）时，sufficient=false，其他字段可留空或填“无”。"),
		schema.SystemMessage("当 force=true 时，即便信息不足，也要尽量给出参与者/起因/经过/结果，缺失部分写“无”。"),
		schema.UserMessage("群:{group_id}"),
		schema.UserMessage("force:{force}"),
		schema.UserMessage("聊天记录:\n{events_text}"),
	)
	return &MemoryExtractor{
		model:    model,
		template: template,
	}
}

func (e *MemoryExtractor) ExtractBatch(ctx context.Context, groupID string, messages []MemoryMessage, force bool) MemoryExtraction {
	if e == nil || e.model == nil || e.template == nil {
		return MemoryExtraction{}
	}
	if len(messages) == 0 {
		return MemoryExtraction{}
	}
	in, err := e.template.Format(ctx, map[string]any{
		"group_id":    groupID,
		"force":       force,
		"events_text": formatMemoryMessages(messages),
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
		if tc.Function.Name != "extract_memory_event" {
			continue
		}
		var result MemoryExtraction
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &result); err != nil {
			llog.Error("解析记忆提取工具参数失败: %v", err)
			break
		}
		result.Cause = strings.TrimSpace(result.Cause)
		result.Process = strings.TrimSpace(result.Process)
		result.Result = strings.TrimSpace(result.Result)
		cleanParticipants := make([]string, 0, len(result.Participants))
		for _, participant := range result.Participants {
			participant = strings.TrimSpace(participant)
			if participant == "" {
				continue
			}
			cleanParticipants = append(cleanParticipants, participant)
		}
		result.Participants = dedupeStrings(cleanParticipants)
		return result
	}
	return MemoryExtraction{}
}

func formatMemoryMessages(messages []MemoryMessage) string {
	lines := make([]string, 0, len(messages))
	for i, msg := range messages {
		speaker := memoryMessageSpeaker(msg)
		if speaker == "" {
			speaker = "用户"
		}
		line := fmt.Sprintf("[%d] %s(%v) %s", i+1, speaker, msg.Role, strings.TrimSpace(msg.Content))
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
