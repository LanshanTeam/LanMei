package default_plugins

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"

	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

const (
	GITHUB_OPEN_GRAPH_URL = "https://opengraph.githubassets.com/%v/%v"
)

var githubURLRegex = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?github\.com/([a-z0-9_.-]+/[a-z0-9_.-]+(?:/[^\s?#]*)?)`)

type GitHubCardPlugin struct{}

func (p *GitHubCardPlugin) Name() string {
	return "GitHub Card 插件"
}

func (p *GitHubCardPlugin) Version() string {
	return "1.0.0"
}

func (p *GitHubCardPlugin) Description() string {
	return "GitHub Card插件，检测到 GitHub 链接会返回卡片预览"
}

func (p *GitHubCardPlugin) Author() string {
	return "Rinai"
}

func (p *GitHubCardPlugin) Enabled() bool {
	return true
}

func (p *GitHubCardPlugin) Trigger(input string, ctx *zero.Ctx) bool {
	return githubURLRegex.MatchString(input)
}

func (p *GitHubCardPlugin) Execute(input string, ctx *zero.Ctx) error {
	matches := githubURLRegex.FindStringSubmatch(input)
	if len(matches) < 2 {
		return nil
	}
	path := strings.TrimRight(matches[1], ".,);]}>\"'")
	cardUrl := fmt.Sprintf(GITHUB_OPEN_GRAPH_URL, rand.Text(), path)
	ctx.Send(message.Message{
		message.Image(cardUrl),
	})
	return nil
}

func (p *GitHubCardPlugin) Initialize() error {
	return nil
}
