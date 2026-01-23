package ba_logo

import (
	"encoding/base64"
	"fmt"
	"os"
	"testing"
)

func TestGetBALOGO(t *testing.T) {
	picbase64 := GetBALOGO("测试", "LanMei")

	fmt.Println(picbase64)
	data, err := base64.StdEncoding.DecodeString(picbase64)
	if err != nil {
		return
	}

	fd, _ := os.OpenFile("balogo_test.png", os.O_CREATE|os.O_WRONLY, 0644)
	fd.Write(data)
}
