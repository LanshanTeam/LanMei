package tts

import (
	"LanMei/bot/utils/llog"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wujunwei928/edge-tts-go/edge_tts"
	"github.com/youthlin/silk"
)

var BasePath = "./data/tts/"

func TTS(text, name string) string {
	if err := os.MkdirAll(BasePath, 0755); err != nil {
		llog.Error("创建 TTS 目录失败:", err)
		return ""
	}

	base := filepath.Join(BasePath, name)
	mp3File := base + ".mp3"
	silkFile := base + ".silk"

	conn, err := edge_tts.NewCommunicate(text,
		edge_tts.SetVoice("zh-CN-XiaoxiaoNeural"),
	)
	if err != nil {
		llog.Error("TTS 连接失败:", err)
		return ""
	}

	audioData, err := conn.Stream()
	if err != nil {
		llog.Error("TTS 流处理失败:", err)
		return ""
	}

	if err = os.WriteFile(mp3File, audioData, 0644); err != nil {
		llog.Error("写入 MP3 文件失败:", err)
		return ""
	}

	mp3ToSilk(mp3File, silkFile)

	_ = os.Remove(mp3File)

	return name + ".silk"
}

// mp3ToSilk 使用 ffmpeg 将 MP3 解码为 PCM，再调用 silk.Encode 生成 SILK
func mp3ToSilk(mp3Path, silkPath string) {
	pcmPath := strings.TrimSuffix(silkPath, filepath.Ext(silkPath)) + ".pcm"

	cmd := exec.Command("ffmpeg",
		"-i", mp3Path,
		"-f", "s16le",
		"-ac", "1",
		"-ar", "24000",
		pcmPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		llog.Error("ffmpeg 转换 PCM 失败:", err, "stderr:", stderr.String())
		return
	}

	f, err := os.Open(pcmPath)
	if err != nil {
		llog.Error("打开 PCM 文件失败:", err)
		_ = os.Remove(pcmPath)
		return
	}
	defer f.Close()

	silkData, err := silk.Encode(f)
	if err != nil {
		llog.Error("SILK 编码失败:", err)
		_ = os.Remove(pcmPath)
		return
	}

	if err = os.Remove(pcmPath); err != nil {
		llog.Error("删除临时 PCM 文件失败:", err)
	}

	if err = os.WriteFile(silkPath, silkData, 0644); err != nil {
		llog.Error("写入 SILK 文件失败:", err)
		return
	}
}
