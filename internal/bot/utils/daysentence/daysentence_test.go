package daysentence

import (
	"fmt"
	"testing"
)

func TestGetDaySentence(t *testing.T) {
	resp := GetDaySentence()
	fmt.Println(fmt.Sprintf("每日一句：%s\n出处：%s\n，作者：%v", resp.Hitokoto, resp.From, resp.FromWho))
}
