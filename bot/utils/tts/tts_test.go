package tts

import (
	"fmt"
	"testing"
)

func TestTTS(t *testing.T) {
	name := TTS("你好，世界！", "test")
	fmt.Println(name)
}
