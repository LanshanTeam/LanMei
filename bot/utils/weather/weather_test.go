package weather

import (
	"fmt"
	"testing"
)

func TestGetWeather(t *testing.T) {
	recs, err := GetWeather("3") // 查未来 3 小时
	if err != nil {
		t.Fatalf("GetWeather 出错: %v", err)
	}
	if len(recs) == 0 {
		t.Fatalf("结果为空")
	}
	fmt.Println(recs)

	text := "蓝妹找到啦～\n"
	for _, v := range recs {
		raw := fmt.Sprintf(
			"时间%s，温度%.1f℃，湿度%d%%，风速%.1f km/h，降水%.1f mm\n",
			v.Time.Format("2006-01-02 15:04"),
			v.Temperature,
			v.Humidity,
			v.WindSpeed,
			v.Precipitation,
		)

		text += raw
	}

	fmt.Println(text)
}
