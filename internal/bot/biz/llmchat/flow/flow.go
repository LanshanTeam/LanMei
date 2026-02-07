package flow

import (
	"context"

	"LanMei/internal/bot/biz/llmchat/flow/nodes"
	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"

	"github.com/cloudwego/eino/compose"
)

type ChatFlow struct {
	runner compose.Runnable[*flowtypes.State, *flowtypes.State]
}

func NewChatFlow(deps flowtypes.Dependencies) (*ChatFlow, error) {
	g := compose.NewGraph[*flowtypes.State, *flowtypes.State]()
	if err := g.AddLambdaNode("init", compose.InvokableLambda(nodes.InitNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("user_context", compose.InvokableLambda(nodes.UserContextNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("analysis", compose.InvokableLambda(nodes.AnalysisNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("judge", compose.InvokableLambda(nodes.JudgeNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("plan", compose.InvokableLambda(nodes.PlanNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("gather_context", compose.InvokableLambda(nodes.GatherContextNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("search_format", compose.InvokableLambda(nodes.SearchFormatNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("build_prompt", compose.InvokableLambda(nodes.BuildPromptNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("chat", compose.InvokableLambda(nodes.ChatNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("post_process", compose.InvokableLambda(nodes.PostProcessNode(deps))); err != nil {
		return nil, err
	}

	if err := g.AddEdge(compose.START, "init"); err != nil {
		return nil, err
	}
	if err := g.AddEdge("init", "user_context"); err != nil {
		return nil, err
	}
	if err := g.AddEdge("user_context", "analysis"); err != nil {
		return nil, err
	}
	if err := g.AddEdge("analysis", "judge"); err != nil {
		return nil, err
	}
	if err := g.AddEdge("judge", "plan"); err != nil {
		return nil, err
	}
	if err := g.AddEdge("plan", "gather_context"); err != nil {
		return nil, err
	}
	if err := g.AddEdge("gather_context", "search_format"); err != nil {
		return nil, err
	}
	if err := g.AddEdge("search_format", "build_prompt"); err != nil {
		return nil, err
	}
	if err := g.AddEdge("build_prompt", "chat"); err != nil {
		return nil, err
	}
	if err := g.AddEdge("chat", "post_process"); err != nil {
		return nil, err
	}
	if err := g.AddEdge("post_process", compose.END); err != nil {
		return nil, err
	}

	runner, err := g.Compile(context.Background(), compose.WithGraphName("lanmei_chat_flow"))
	if err != nil {
		return nil, err
	}
	return &ChatFlow{runner: runner}, nil
}

func (f *ChatFlow) Run(ctx context.Context, req flowtypes.Request) (string, error) {
	if f == nil || f.runner == nil {
		return "", nil
	}
	state := flowtypes.NewState(req)
	out, err := f.runner.Invoke(ctx, state)
	if err != nil {
		return "", err
	}
	if out == nil {
		return "", nil
	}
	return out.Reply, nil
}
