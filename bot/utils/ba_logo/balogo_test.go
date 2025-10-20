package ba_logo

import (
	"os"
	"testing"
)

func TestGetBALOGO(t *testing.T) {
	data := GetBALOGO("123", "456")

	fd, _ := os.OpenFile("balogo_test.png", os.O_CREATE|os.O_WRONLY, 0644)
	fd.Write(data)
}
