package react

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"LanMei/internal/bot/biz/llmchat/reactlog"
	"LanMei/internal/bot/utils/llog"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	skillsmw "github.com/dyike/eino-skills/pkg/middleware"
	skillpkg "github.com/dyike/eino-skills/pkg/skill"
	skilltools "github.com/dyike/eino-skills/pkg/tools"
)

func InitSkills() (*skillpkg.Registry, *skillsmw.SkillsMiddleware, []tool.BaseTool) {
	loader := skillpkg.NewLoader(skillpkg.WithGlobalSkillsDir("skills"))
	registry := skillpkg.NewRegistry(loader)
	if registry == nil {
		llog.Error("初始化技能注册表失败")
		return nil, nil, nil
	}
	if err := registry.Initialize(context.Background()); err != nil {
		llog.Error("初始化技能失败: %v", err)
	}
	middleware := skillsmw.NewSkillsMiddleware(registry)
	tools := skilltools.NewSkillTools(registry)
	return registry, middleware, tools
}

func TraceToolMiddleware(trace *reactlog.Window) compose.ToolMiddleware {
	if trace == nil {
		return compose.ToolMiddleware{}
	}
	return compose.ToolMiddleware{
		Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
			return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
				groupID := reactlog.GroupIDFromContext(ctx)
				if groupID != "" && input != nil {
					args := strings.TrimSpace(input.Arguments)
					if args == "" {
						args = "{}"
					}
					trace.Append(groupID, reactlog.StepAction, fmt.Sprintf("%s %s", input.Name, args))
				}
				out, err := next(ctx, input)
				if err != nil {
					return out, err
				}
				if groupID := reactlog.GroupIDFromContext(ctx); groupID != "" && out != nil {
					trace.Append(groupID, reactlog.StepObservation, out.Result)
				}
				return out, nil
			}
		},
	}
}

func BuildUnknownToolHandler(tools []tool.BaseTool, trace *reactlog.Window) func(ctx context.Context, name, input string) (string, error) {
	invokers := map[string]tool.InvokableTool{}
	for _, base := range tools {
		invokable, ok := base.(tool.InvokableTool)
		if !ok {
			continue
		}
		info, err := invokable.Info(context.Background())
		if err != nil || info == nil || strings.TrimSpace(info.Name) == "" {
			continue
		}
		invokers[info.Name] = invokable
	}
	return func(ctx context.Context, name, input string) (string, error) {
		toolName, argsJSON, ok := normalizeToolCall(name, input, invokers)
		if !ok {
			return fmt.Sprintf("未知工具: %s", name), nil
		}
		if trace != nil {
			if groupID := reactlog.GroupIDFromContext(ctx); groupID != "" {
				args := strings.TrimSpace(argsJSON)
				if args == "" {
					args = "{}"
				}
				trace.Append(groupID, reactlog.StepAction, fmt.Sprintf("%s %s", toolName, args))
			}
		}
		output, err := invokers[toolName].InvokableRun(ctx, argsJSON)
		if err != nil {
			return "", err
		}
		if trace != nil {
			if groupID := reactlog.GroupIDFromContext(ctx); groupID != "" && strings.TrimSpace(output) != "" {
				trace.Append(groupID, reactlog.StepObservation, output)
			}
		}
		return output, nil
	}
}

func normalizeToolCall(name, input string, invokers map[string]tool.InvokableTool) (string, string, bool) {
	name = strings.TrimSpace(name)
	input = strings.TrimSpace(input)
	if name == "" {
		return "", "", false
	}
	if _, ok := invokers[name]; ok {
		if input != "" && json.Valid([]byte(input)) {
			return name, input, true
		}
		if input == "" {
			return name, "{}", true
		}
		return "", "", false
	}

	for toolName := range invokers {
		if !strings.HasPrefix(name, toolName) {
			continue
		}
		remainder := strings.TrimPrefix(name, toolName)
		args := parseArgsFromName(toolName, remainder)
		if len(args) == 0 {
			if input != "" && json.Valid([]byte(input)) && strings.TrimSpace(input) != "{}" {
				return toolName, input, true
			}
			if toolName == "list_skills" {
				return toolName, "{}", true
			}
			return "", "", false
		}
		payload, err := json.Marshal(args)
		if err != nil {
			return "", "", false
		}
		return toolName, string(payload), true
	}

	return "", "", false
}

func parseArgsFromName(toolName, raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if value := extractLooseTagValue(raw, "arg_value"); value != "" {
		key := extractLooseTagValue(raw, "arg_key")
		if key == "" {
			key = defaultArgKey(toolName)
		}
		if key != "" {
			return map[string]string{key: value}
		}
	}
	if key, value := parseKeyValue(raw); key != "" && value != "" {
		return map[string]string{key: value}
	}
	if fallback := defaultArgKey(toolName); fallback != "" {
		raw = strings.Trim(raw, " _()")
		raw = strings.Trim(raw, "\"'")
		if raw != "" {
			return map[string]string{fallback: raw}
		}
	}
	return nil
}

func extractLooseTagValue(raw, tag string) string {
	if value := extractTagValue(raw, tag); value != "" {
		return value
	}
	needle := tag + ">"
	start := strings.Index(raw, needle)
	if start == -1 {
		return ""
	}
	start += len(needle)
	endTag := "</" + tag + ">"
	end := strings.Index(raw[start:], endTag)
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(raw[start : start+end])
}

func extractTagValue(raw, tag string) string {
	startTag := "<" + tag + ">"
	endTag := "</" + tag + ">"
	start := strings.Index(raw, startTag)
	if start == -1 {
		return ""
	}
	start += len(startTag)
	end := strings.Index(raw[start:], endTag)
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(raw[start : start+end])
}

func parseKeyValue(raw string) (string, string) {
	raw = strings.Trim(raw, " _()")
	if raw == "" {
		return "", ""
	}
	if idx := strings.IndexAny(raw, "=:"); idx >= 0 {
		key := strings.TrimSpace(raw[:idx])
		value := strings.TrimSpace(raw[idx+1:])
		key = strings.Trim(key, "\"'")
		value = strings.Trim(value, "\"' )")
		return key, value
	}
	return "", ""
}

func defaultArgKey(toolName string) string {
	switch toolName {
	case "view_skill":
		return "name"
	case "web_search":
		return "query"
	case "recall_memory", "recall_knowledge":
		return "query"
	default:
		return ""
	}
}
