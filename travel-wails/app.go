package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Location holds geocoded location data.
type Location struct {
	Name     string  `json:"name"`
	Country  string  `json:"country"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Timezone string  `json:"timezone"`
}

// DayForecast holds forecast data for one day.
type DayForecast struct {
	Date      string  `json:"date"`
	Icon      string  `json:"icon"`
	Condition string  `json:"condition"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Precip    int     `json:"precip"`
}

// WeatherResult combines location and forecast.
type WeatherResult struct {
	Location Location      `json:"location"`
	Forecast []DayForecast `json:"forecast"`
}

// GetWeather geocodes a city and returns a 7-day forecast.
func (a *App) GetWeather(city string) (*WeatherResult, error) {
	// --- Geocode ---
	geoURL := fmt.Sprintf(
		"https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1&language=en&format=json",
		url.QueryEscape(city))
	resp, err := http.Get(geoURL)
	if err != nil {
		return nil, fmt.Errorf("geocoding request failed: %w", err)
	}
	defer resp.Body.Close()
	geoBody, _ := io.ReadAll(resp.Body)

	var geoData struct {
		Results []struct {
			Name     string  `json:"name"`
			Country  string  `json:"country"`
			Lat      float64 `json:"latitude"`
			Lon      float64 `json:"longitude"`
			Timezone string  `json:"timezone"`
		} `json:"results"`
	}
	if err := json.Unmarshal(geoBody, &geoData); err != nil {
		return nil, fmt.Errorf("geocoding parse error: %w", err)
	}
	if len(geoData.Results) == 0 {
		return nil, fmt.Errorf("location not found: %s", city)
	}
	g := geoData.Results[0]
	loc := Location{
		Name:     g.Name,
		Country:  g.Country,
		Lat:      g.Lat,
		Lon:      g.Lon,
		Timezone: g.Timezone,
	}

	// --- Fetch forecast ---
	forecastURL := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f"+
			"&daily=temperature_2m_max,temperature_2m_min,weather_code,precipitation_probability_max,wind_speed_10m_max"+
			"&forecast_days=7&timezone=auto",
		g.Lat, g.Lon)
	wresp, err := http.Get(forecastURL)
	if err != nil {
		return nil, fmt.Errorf("forecast request failed: %w", err)
	}
	defer wresp.Body.Close()
	wbody, _ := io.ReadAll(wresp.Body)

	var wdata struct {
		Daily struct {
			Time        []string  `json:"time"`
			TempMax     []float64 `json:"temperature_2m_max"`
			TempMin     []float64 `json:"temperature_2m_min"`
			WeatherCode []int     `json:"weather_code"`
			PrecipProb  []int     `json:"precipitation_probability_max"`
		} `json:"daily"`
	}
	if err := json.Unmarshal(wbody, &wdata); err != nil {
		return nil, fmt.Errorf("forecast parse error: %w", err)
	}

	n := len(wdata.Daily.Time)
	forecast := make([]DayForecast, n)
	for i := 0; i < n; i++ {
		code := wdata.Daily.WeatherCode[i]
		precip := 0
		if i < len(wdata.Daily.PrecipProb) {
			precip = wdata.Daily.PrecipProb[i]
		}
		forecast[i] = DayForecast{
			Date:      wdata.Daily.Time[i],
			Icon:      wmoEmoji(code),
			Condition: wmoDescription(code),
			High:      wdata.Daily.TempMax[i],
			Low:       wdata.Daily.TempMin[i],
			Precip:    precip,
		}
	}

	return &WeatherResult{Location: loc, Forecast: forecast}, nil
}

func wmoDescription(code int) string {
	switch {
	case code == 0:
		return "Clear sky"
	case code <= 3:
		return "Partly cloudy"
	case code <= 49:
		return "Fog"
	case code <= 59:
		return "Drizzle"
	case code <= 69:
		return "Rain"
	case code <= 79:
		return "Snow"
	case code <= 84:
		return "Rain showers"
	case code <= 86:
		return "Snow showers"
	case code <= 99:
		return "Thunderstorm"
	default:
		return "Unknown"
	}
}

func wmoEmoji(code int) string {
	switch {
	case code == 0:
		return "☀️"
	case code <= 3:
		return "⛅"
	case code <= 49:
		return "🌫️"
	case code <= 59:
		return "🌦️"
	case code <= 69:
		return "🌧️"
	case code <= 79:
		return "🌨️"
	case code <= 84:
		return "🌧️"
	case code <= 86:
		return "🌨️"
	case code <= 99:
		return "⛈️"
	default:
		return "❓"
	}
}
