package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// geocodingResult holds a location from the Open-Meteo geocoding API.
type geocodingResult struct {
	Name     string  `json:"name"`
	Country  string  `json:"country"`
	Lat      float64 `json:"latitude"`
	Lon      float64 `json:"longitude"`
	Timezone string  `json:"timezone"`
}

// weatherResponse holds the Open-Meteo forecast response.
type weatherResponse struct {
	Daily struct {
		Time         []string  `json:"time"`
		TempMax      []float64 `json:"temperature_2m_max"`
		TempMin      []float64 `json:"temperature_2m_min"`
		WeatherCode  []int     `json:"weather_code"`
		PrecipProb   []int     `json:"precipitation_probability_max"`
		WindSpeedMax []float64 `json:"wind_speed_10m_max"`
	} `json:"daily"`
}

// wmoDescription maps WMO weather codes to short descriptions.
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

// wmoEmoji maps WMO weather codes to an emoji-like symbol.
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

func geocode(city string) (*geocodingResult, error) {
	u := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1&language=en&format=json",
		url.QueryEscape(city))
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var data struct {
		Results []geocodingResult `json:"results"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if len(data.Results) == 0 {
		return nil, fmt.Errorf("location not found: %s", city)
	}
	return &data.Results[0], nil
}

func fetchWeather(lat, lon float64) (*weatherResponse, error) {
	u := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f"+
			"&daily=temperature_2m_max,temperature_2m_min,weather_code,precipitation_probability_max,wind_speed_10m_max"+
			"&forecast_days=7&timezone=auto",
		lat, lon)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var w weatherResponse
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

func main() {
	a := app.New()
	w := a.NewWindow("Travel Weather")
	w.Resize(fyne.NewSize(520, 500))

	// --- Input row ---
	cityEntry := widget.NewEntry()
	cityEntry.SetPlaceHolder("Enter city (e.g. Tokyo)")

	// --- Status / clock labels ---
	locationLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	clockLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Monospace: true})
	statusLabel := widget.NewLabelWithStyle("Enter a city and press Search", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

	// --- Forecast table (hidden until data loads) ---
	forecastData := &forecastTable{}
	table := widget.NewTable(
		func() (int, int) { return forecastData.Rows(), 6 },
		func() fyne.CanvasObject { return widget.NewLabel("placeholder text") },
		func(id widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			label.SetText(forecastData.Cell(id.Row, id.Col))
			if id.Row == 0 {
				label.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				label.TextStyle = fyne.TextStyle{}
			}
			label.Refresh()
		},
	)
	table.SetColumnWidth(0, 100) // Date
	table.SetColumnWidth(1, 50)  // Icon
	table.SetColumnWidth(2, 110) // Condition
	table.SetColumnWidth(3, 80)  // High
	table.SetColumnWidth(4, 80)  // Low
	table.SetColumnWidth(5, 80)  // Precip %
	table.Hide()

	// --- Clock ticker state ---
	var tickerStop chan struct{}
	var tz *time.Location

	startClock := func(timezone string) {
		if tickerStop != nil {
			close(tickerStop)
		}
		var err error
		tz, err = time.LoadLocation(timezone)
		if err != nil {
			tz = time.UTC
		}
		tickerStop = make(chan struct{})
		go func(stop chan struct{}) {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					now := time.Now().In(tz)
					clockLabel.SetText(now.Format("Monday, 02 Jan 2006  15:04:05 MST"))
				}
			}
		}(tickerStop)
	}

	// --- Search action ---
	doSearch := func() {
		city := strings.TrimSpace(cityEntry.Text)
		if city == "" {
			statusLabel.SetText("Please enter a city name.")
			return
		}
		statusLabel.SetText("Searching...")
		locationLabel.SetText("")
		clockLabel.SetText("")
		table.Hide()

		go func() {
			loc, err := geocode(city)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				return
			}

			weather, err := fetchWeather(loc.Lat, loc.Lon)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				return
			}

			locationLabel.SetText(fmt.Sprintf("%s, %s  (%.2f°, %.2f°)", loc.Name, loc.Country, loc.Lat, loc.Lon))
			startClock(loc.Timezone)

			forecastData.Load(weather)
			table.Show()
			table.Refresh()
			statusLabel.SetText(fmt.Sprintf("7-day forecast for %s", loc.Name))
		}()
	}

	searchBtn := widget.NewButtonWithIcon("Search", theme.SearchIcon(), func() { doSearch() })
	cityEntry.OnSubmitted = func(_ string) { doSearch() }

	inputRow := container.NewBorder(nil, nil, nil, searchBtn, cityEntry)

	content := container.NewVBox(
		inputRow,
		widget.NewSeparator(),
		locationLabel,
		clockLabel,
		widget.NewSeparator(),
		statusLabel,
		container.New(layout.NewStackLayout(), table),
	)

	w.SetContent(container.NewPadded(content))
	w.ShowAndRun()
}

// forecastTable provides data for the table widget.
type forecastTable struct {
	dates      []string
	icons      []string
	conditions []string
	highs      []string
	lows       []string
	precips    []string
}

func (f *forecastTable) Rows() int {
	if len(f.dates) == 0 {
		return 0
	}
	return len(f.dates) + 1 // +1 for header
}

func (f *forecastTable) Load(w *weatherResponse) {
	n := len(w.Daily.Time)
	f.dates = make([]string, n)
	f.icons = make([]string, n)
	f.conditions = make([]string, n)
	f.highs = make([]string, n)
	f.lows = make([]string, n)
	f.precips = make([]string, n)
	for i := 0; i < n; i++ {
		f.dates[i] = w.Daily.Time[i]
		f.icons[i] = wmoEmoji(w.Daily.WeatherCode[i])
		f.conditions[i] = wmoDescription(w.Daily.WeatherCode[i])
		f.highs[i] = fmt.Sprintf("%.1f °C", w.Daily.TempMax[i])
		f.lows[i] = fmt.Sprintf("%.1f °C", w.Daily.TempMin[i])
		if i < len(w.Daily.PrecipProb) {
			f.precips[i] = fmt.Sprintf("%d%%", w.Daily.PrecipProb[i])
		}
	}
}

func (f *forecastTable) Cell(row, col int) string {
	if row == 0 {
		return [...]string{"Date", "", "Condition", "High", "Low", "Precip"}[col]
	}
	i := row - 1
	switch col {
	case 0:
		return f.dates[i]
	case 1:
		return f.icons[i]
	case 2:
		return f.conditions[i]
	case 3:
		return f.highs[i]
	case 4:
		return f.lows[i]
	case 5:
		return f.precips[i]
	}
	return ""
}
