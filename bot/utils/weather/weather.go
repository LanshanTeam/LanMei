package weather

import (
	"LanMei/bot/utils/llog"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

var baseURL = "https://api.open-meteo.com/v1/forecast?" +
	"latitude=29.56&longitude=106.55" +
	"&hourly=temperature_2m,precipitation,relative_humidity_2m,wind_speed_10m" +
	"&forecast_hours=%d" +
	"&timezone=Asia%%2FShanghai"

func mustSameLen[T any](name string, base int, arr []T) error {
	if len(arr) != base {
		return errors.New("与time长度不一致: " + name)
	}
	return nil
}

func GetWeather(hour string) ([]HourRecord, error) {
	client := &http.Client{}
	h, err := strconv.Atoi(hour)
	if h <= 0 || h > 8 {
		return nil, errors.New("hour参数必须在1到8之间")
	}
	if err != nil {
		llog.Error(err.Error())
		return nil, err
	}

	url := fmt.Sprintf(baseURL, h)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		llog.Error(err.Error())
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		llog.Error(err.Error())
		return nil, err
	}
	defer res.Body.Close()
	resp, err := io.ReadAll(res.Body)
	if err != nil {
		llog.Error(err.Error())
		return nil, err
	}

	var f Forecast
	if err = json.Unmarshal(resp, &f); err != nil {
		llog.Error(err.Error())
		return nil, err
	}

	n := len(f.Hourly.Time)

	err = mustSameLen("temperature_2m", n, f.Hourly.Temperature2m)
	err = mustSameLen("precipitation", n, f.Hourly.Precipitation)
	err = mustSameLen("relative_humidity_2m", n, f.Hourly.RelativeHumidity2m)
	err = mustSameLen("wind_speed_10m", n, f.Hourly.WindSpeed10m)
	if err != nil {
		llog.Error(err.Error())
		return nil, err
	}

	loc, _ := time.LoadLocation("Asia/Shanghai")
	rec := make([]HourRecord, 0, n)
	for i := 0; i < n; i++ {
		t, err := time.ParseInLocation("2006-01-02T15:04", f.Hourly.Time[i], loc)
		if err != nil {
			llog.Error(err.Error())
			return nil, err
		}
		rec = append(rec, HourRecord{
			Time:          t.In(loc),
			Temperature:   f.Hourly.Temperature2m[i],
			Precipitation: f.Hourly.Precipitation[i],
			Humidity:      f.Hourly.RelativeHumidity2m[i],
			WindSpeed:     f.Hourly.WindSpeed10m[i],
		})
	}
	return rec, nil
}
