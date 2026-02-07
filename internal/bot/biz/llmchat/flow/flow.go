package flow

import (
	"context"

	"github.com/cloudwego/eino/compose"
)

type ChatFlow struct {
	runner compose.Runnable[*State, *State]
}

func NewChatFlow(deps Dependencies) (*ChatFlow, error) {
	g := compose.NewGraph[*State, *State]()
	if err := g.AddLambdaNode("init", compose.InvokableLambda(initNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("user_context", compose.InvokableLambda(userContextNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("analysis", compose.InvokableLambda(analysisNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("judge", compose.InvokableLambda(judgeNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("plan", compose.InvokableLambda(planNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("gather_context", compose.InvokableLambda(gatherContextNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("search_format", compose.InvokableLambda(searchFormatNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("build_prompt", compose.InvokableLambda(buildPromptNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("chat", compose.InvokableLambda(chatNode(deps))); err != nil {
		return nil, err
	}
	if err := g.AddLambdaNode("post_process", compose.InvokableLambda(postProcessNode(deps))); err != nil {
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

func (f *ChatFlow) Run(ctx context.Context, req Request) (string, error) {
	if f == nil || f.runner == nil {
		return "", nil
	}
	state := NewState(req)
	out, err := f.runner.Invoke(ctx, state)
	if err != nil {
		return "", err
	}
	if out == nil {
		return "", nil
	}
	return out.Reply, nil
}
