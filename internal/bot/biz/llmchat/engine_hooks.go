package llmchat

import "LanMei/internal/bot/biz/llmchat/flow/hooks"

// AddHook registers an additional LLM hook for this engine instance.
func (c *ChatEngine) AddHook(h hooks.Hook) {
	if c == nil || c.hooks == nil {
		return
	}
	c.hooks.Add(h)
}
