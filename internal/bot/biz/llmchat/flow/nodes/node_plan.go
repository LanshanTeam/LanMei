package nodes

import (
	"context"

	flowtypes "LanMei/internal/bot/biz/llmchat/flow/types"
)

func PlanNode(deps flowtypes.Dependencies) func(context.Context, *flowtypes.State) (*flowtypes.State, error) {
	return func(ctx context.Context, state *flowtypes.State) (*flowtypes.State, error) {
		if state == nil || state.Stop {
			return state, nil
		}
		plan := buildPlan(ctx, deps, state)
		if plan.Action == "" {
			state.StopWith("plan_empty")
			return state, nil
		}
		if plan.Action == "wait" && !state.Request.Must {
			state.StopWith("plan_wait")
			return state, nil
		}
		if plan.Action == "ask_clarify" {
			plan.NeedClarify = true
		}
		state.Plan = plan
		return state, nil
	}
}
