package default_plugins

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"LanMei/internal/bot/utils/llog"
	"LanMei/internal/bot/utils/netease"

	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

const (
	musicCommand = "/music"
)

type musicSession struct {
	Songs    []netease.SongInfo
	ExpireAt time.Time
}

type MusicCardPlugin struct {
	sync.Mutex
	sessions map[string]musicSession
	Client   *netease.Client
}

func (m *MusicCardPlugin) Name() string {
	return "音乐卡片插件"
}

func (m *MusicCardPlugin) Description() string {
	return "根据用户的指令生成音乐卡片，展示歌曲信息和封面。"
}

func (m *MusicCardPlugin) Author() string {
	return "Rinai"
}

func (m *MusicCardPlugin) Version() string {
	return "1.0.0"
}

func (m *MusicCardPlugin) Enabled() bool {
	return true
}

func (m *MusicCardPlugin) Trigger(input string, ctx *zero.Ctx) bool {
	if strings.HasPrefix(input, musicCommand) {
		return true
	}
	return isSelectionInput(input) && m.hasActiveSession(sessionKey(ctx))
}

func (m *MusicCardPlugin) Execute(input string, ctx *zero.Ctx) error {
	// 清理过期的会话
	m.cleanupSessions()

	if strings.HasPrefix(input, musicCommand) {
		return m.handleSearch(input, ctx)
	}
	if isSelectionInput(input) {
		return m.handleSelection(input, ctx)
	}
	return nil
}

func (m *MusicCardPlugin) Initialize() error {
	if m.sessions == nil {
		m.sessions = make(map[string]musicSession)
	}
	m.Client = netease.NewClient()
	PluginInitializeLog(m)
	return nil
}

// /music [搜索] 触发搜索流程，展示歌曲列表
// 用户回复序号触发选择流程，展示音乐卡片
func (m *MusicCardPlugin) handleSearch(input string, ctx *zero.Ctx) error {
	searchQuery := strings.TrimSpace(strings.TrimPrefix(input, musicCommand))
	if searchQuery == "" {
		ctx.Send(message.Message{
			message.Text("请输入歌曲名，例如：/music 星降る海"),
		})
		return nil
	}

	songs, err := m.Client.SearchSongs(searchQuery, 3)
	if err != nil {
		llog.Error("music search failed: %v", err)
		ctx.Send(message.Message{
			message.Text("搜索失败，请稍后再试。"),
		})
		return nil
	}
	if len(songs) == 0 {
		ctx.Send(message.Message{
			message.Text("未找到相关音乐。"),
		})
		return nil
	}

	lines := []string{fmt.Sprintf("找到 %d 首歌曲，请回复序号选择：", len(songs))}
	for i, song := range songs {
		artists := "未知歌手"
		if len(song.Artists) > 0 {
			artists = strings.Join(song.Artists, " / ")
		}
		album := song.Album
		if album == "" {
			album = "未知专辑"
		}
		lines = append(lines, fmt.Sprintf("%d. %s - %s 《%s》 [%s]", i+1, song.Name, artists, album, formatDuration(song.DurationMs)))
	}

	key := sessionKey(ctx)
	m.saveSession(key, songs, 60*time.Second)
	ctx.Send(message.Message{
		message.Text(strings.Join(lines, "\n")),
	})
	return nil
}

// 处理用户选择，展示音乐卡片
func (m *MusicCardPlugin) handleSelection(input string, ctx *zero.Ctx) error {
	key := sessionKey(ctx)
	session, ok := m.getSession(key)
	if !ok {
		return nil
	}
	if time.Now().After(session.ExpireAt) {
		m.deleteSession(key)
		return nil
	}

	index, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || index <= 0 || index > len(session.Songs) {
		return nil
	}

	selected := session.Songs[index-1]
	ctx.Send(message.Message{
		message.Music("163", selected.ID),
	})
	m.deleteSession(key)
	return nil
}

func isSelectionInput(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return false
	}
	_, err := strconv.Atoi(input)
	return err == nil
}

func sessionKey(ctx *zero.Ctx) string {
	return fmt.Sprintf("%d:%d", ctx.Event.GroupID, ctx.Event.UserID)
}

func (m *MusicCardPlugin) saveSession(key string, songs []netease.SongInfo, ttl time.Duration) {
	m.Lock()
	defer m.Unlock()
	if m.sessions == nil {
		m.sessions = map[string]musicSession{}
	}
	m.sessions[key] = musicSession{
		Songs:    songs,
		ExpireAt: time.Now().Add(ttl),
	}
}

func (m *MusicCardPlugin) getSession(key string) (musicSession, bool) {
	m.Lock()
	defer m.Unlock()
	session, ok := m.sessions[key]
	return session, ok
}

func (m *MusicCardPlugin) deleteSession(key string) {
	m.Lock()
	defer m.Unlock()
	delete(m.sessions, key)
}

func (m *MusicCardPlugin) hasActiveSession(key string) bool {
	m.Lock()
	defer m.Unlock()
	session, ok := m.sessions[key]
	if !ok {
		return false
	}
	if time.Now().After(session.ExpireAt) {
		delete(m.sessions, key)
		return false
	}
	return true
}

func (m *MusicCardPlugin) cleanupSessions() {
	m.Lock()
	defer m.Unlock()
	now := time.Now()
	for key, session := range m.sessions {
		if now.After(session.ExpireAt) {
			delete(m.sessions, key)
		}
	}
}

func formatDuration(ms int64) string {
	if ms <= 0 {
		return "未知"
	}
	seconds := ms / 1000
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}
