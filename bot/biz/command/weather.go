package command

import (
	"LanMei/bot/utils/weather"
	"fmt"
)

func Weather(hour string) string {
	rec, err := weather.GetWeather(hour)
	if err != nil {
		return "呜呜～蓝妹获取天气失败了"
	}

	text := "蓝妹找到啦～\n"
	for _, v := range rec {
		raw := fmt.Sprintf(
			"时间 %s，温度 %.1f℃，湿度 %d%%，风速 %.1f km/h，降水 %.1f mm\n",
			v.Time.Format("2006-01-02 15:04"),
			v.Temperature,
			v.Humidity,
			v.WindSpeed,
			v.Precipitation,
		)

		text += raw
	}

	return text
}
