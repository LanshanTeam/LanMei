package default_plugins

import "LanMei/internal/bot/utils/llog"

type pluginMeta interface {
	Name() string
	Version() string
	Description() string
	Author() string
	Enabled() bool
}

func PluginInitializeLog(p pluginMeta) {
	llog.Infof(
		"加载插件 name=%s version=%s author=%s enabled=%t desc=%s",
		p.Name(),
		p.Version(),
		p.Author(),
		p.Enabled(),
		p.Description(),
	)
}
