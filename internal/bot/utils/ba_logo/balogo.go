package ba_logo

import (
	"fmt"
)

var BaLogoURL = "https://balogo.huankong.top/?textL=%v&textR=%v"

func GetBALOGO(left, right string) string {
	return fmt.Sprintf(BaLogoURL, left, right)
}
