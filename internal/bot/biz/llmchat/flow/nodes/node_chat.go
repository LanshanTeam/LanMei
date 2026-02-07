package nodes

import (
	"context"

	"LanMei/internal/bot/biz/llmchat/flow/hooks"
	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
	"LanMei/internal/bot/utils/llog"

	"github.com/cloudwego/eino/schema"
)

func ChatNode(deps flowtypes.Dependencies) func(context.Context, *flowtypes.State) (*flowtypes.State, error) {
	return func(ctx context.Context, state *flowtypes.State) (*flowtypes.State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		if deps.ChatModel == nil {
			state.StopWith("chat_unavailable")
			return state, nil
		}
		msg, err := hooks.Run(ctx, deps.Hooks, deps.HookInfos.Chat, func() (*schema.Message, error) {
			return deps.ChatModel.Generate(ctx, state.Prompt)
		})
		if err != nil {
			llog.Error("generate message error: %v", err)
			state.Reply = state.Request.Input
			state.StopWith("chat_generate_error")
			return state, nil
		}
		llog.Info("消耗 Completion Tokens: ", msg.ResponseMeta.Usage.CompletionTokens)
		llog.Info("消耗 Prompt Tokens: ", msg.ResponseMeta.Usage.PromptTokens)
		llog.Info("消耗 Total Tokens: ", msg.ResponseMeta.Usage.TotalTokens)
		llog.Info("输出消息为：", msg.Content)
		state.Reply = msg.Content
		return state, nil
	}
}
