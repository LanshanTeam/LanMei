package weather

import "time"

type HourlyUnits struct {
	Time               string `json:"time"`
	Temperature2m      string `json:"temperature_2m"`
	Precipitation      string `json:"precipitation"`
	RelativeHumidity2m string `json:"relative_humidity_2m"`
	WindSpeed10m       string `json:"wind_speed_10m"`
}

type Hourly struct {
	Time               []string  `json:"time"`
	Temperature2m      []float64 `json:"temperature_2m"`
	Precipitation      []float64 `json:"precipitation"`
	RelativeHumidity2m []int     `json:"relative_humidity_2m"`
	WindSpeed10m       []float64 `json:"wind_speed_10m"`
}

type Forecast struct {
	Latitude         float64     `json:"latitude"`
	Longitude        float64     `json:"longitude"`
	UtcOffsetSeconds int         `json:"utc_offset_seconds"`
	Timezone         string      `json:"timezone"`
	TimezoneAbbr     string      `json:"timezone_abbreviation"`
	Elevation        float64     `json:"elevation"`
	HourlyUnits      HourlyUnits `json:"hourly_units"`
	Hourly           Hourly      `json:"hourly"`
	GenerationtimeMS float64     `json:"generationtime_ms"`
}

type HourRecord struct {
	Time          time.Time
	Temperature   float64
	Precipitation float64
	Humidity      int
	WindSpeed     float64
}
