// travel-walk is a small Windows GUI app built with github.com/lxn/walk
// that shows the local time and a 7-day weather forecast for a remote
// city. Weather data is fetched from the free Open-Meteo APIs:
//
//	https://geocoding-api.open-meteo.com
//	https://api.open-meteo.com
//
// Build (Windows):
//
//	go build -ldflags="-H windowsgui"
//
// The checked-in rsrc_windows_amd64.syso embeds app.manifest so the
// Common Controls 6.0 library is loaded at startup; without it walk
// crashes during widget init with "TTM_ADDTOOL failed". To regenerate
// the syso after editing app.manifest:
//
//	go install github.com/akavel/rsrc@latest
//	go generate ./...
//
//go:generate rsrc -manifest app.manifest -o rsrc_windows_amd64.syso
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
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
	Current struct {
		Temperature2m float64 `json:"temperature_2m"`
		WeatherCode   int     `json:"weather_code"`
		WindSpeed10m  float64 `json:"wind_speed_10m"`
	} `json:"current"`
	Daily struct {
		Time         []string  `json:"time"`
		TempMax      []float64 `json:"temperature_2m_max"`
		TempMin      []float64 `json:"temperature_2m_min"`
		WeatherCode  []int     `json:"weather_code"`
		PrecipProb   []int     `json:"precipitation_probability_max"`
		WindSpeedMax []float64 `json:"wind_speed_10m_max"`
	} `json:"daily"`
}

// wmoDescription maps a WMO weather code to a short description.
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
	}
	return "Unknown"
}

func geocode(city string) (*geocodingResult, error) {
	u := fmt.Sprintf(
		"https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1&language=en&format=json",
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
			"&current=temperature_2m,weather_code,wind_speed_10m"+
			"&daily=temperature_2m_max,temperature_2m_min,weather_code,"+
			"precipitation_probability_max,wind_speed_10m_max"+
			"&temperature_unit=fahrenheit"+
			"&wind_speed_unit=mph"+
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

// formatLat renders a latitude with a N/S suffix, e.g. "51.51°N".
func formatLat(lat float64) string {
	dir := "N"
	if lat < 0 {
		dir = "S"
		lat = -lat
	}
	return fmt.Sprintf("%.2f°%s", lat, dir)
}

// formatLon renders a longitude with an E/W suffix, e.g. "0.13°W".
func formatLon(lon float64) string {
	dir := "E"
	if lon < 0 {
		dir = "W"
		lon = -lon
	}
	return fmt.Sprintf("%.2f°%s", lon, dir)
}

// ForecastItem is one row in the forecast TableView.
type ForecastItem struct {
	Day       string
	Date      string
	Condition string
	High      string
	Low       string
	Precip    string
	Wind      string
}

// ForecastModel implements walk.TableModel for the forecast TableView.
type ForecastModel struct {
	walk.TableModelBase
	items []*ForecastItem
}

func NewForecastModel() *ForecastModel {
	return &ForecastModel{}
}

func (m *ForecastModel) RowCount() int {
	return len(m.items)
}

func (m *ForecastModel) Value(row, col int) interface{} {
	it := m.items[row]
	switch col {
	case 0:
		return it.Day
	case 1:
		return it.Date
	case 2:
		return it.Condition
	case 3:
		return it.High
	case 4:
		return it.Low
	case 5:
		return it.Precip
	case 6:
		return it.Wind
	}
	return ""
}

func (m *ForecastModel) Load(w *weatherResponse, tz *time.Location) {
	todayYMD := time.Now().In(tz).Format("2006-01-02")

	n := len(w.Daily.Time)
	items := make([]*ForecastItem, n)
	for i := 0; i < n; i++ {
		raw := w.Daily.Time[i]
		var day, date string
		if t, err := time.ParseInLocation("2006-01-02", raw, tz); err != nil {
			day = raw
		} else {
			if raw == todayYMD {
				day = "Today"
			} else {
				day = t.Format("Mon")
			}
			date = t.Format("1/2")
		}

		it := &ForecastItem{
			Day:       day,
			Date:      date,
			Condition: wmoDescription(w.Daily.WeatherCode[i]),
			High:      fmt.Sprintf("%.1f °F", w.Daily.TempMax[i]),
			Low:       fmt.Sprintf("%.1f °F", w.Daily.TempMin[i]),
		}
		if i < len(w.Daily.PrecipProb) {
			it.Precip = fmt.Sprintf("%d%%", w.Daily.PrecipProb[i])
		}
		if i < len(w.Daily.WindSpeedMax) {
			it.Wind = fmt.Sprintf("%.1f mph", w.Daily.WindSpeedMax[i])
		}
		items[i] = it
	}
	m.items = items
	m.PublishRowsReset()
}

func main() {
	var (
		mw           *walk.MainWindow
		cityEdit     *walk.LineEdit
		locLabel     *walk.Label
		coordsLabel  *walk.Label
		currentLabel *walk.Label
		clockLabel   *walk.Label
		statusLabel  *walk.Label
	)

	model := NewForecastModel()

	// tickerStop lets us cancel the previous clock goroutine when the
	// user searches for a new city.
	var tickerStop chan struct{}

	startClock := func(tz *time.Location) {
		if tickerStop != nil {
			close(tickerStop)
		}
		stop := make(chan struct{})
		tickerStop = stop
		go func(stop chan struct{}, tz *time.Location) {
			update := func() {
				text := time.Now().In(tz).Format(
					"Monday 02 Jan 2006  3:04 PM MST")
				mw.Synchronize(func() { clockLabel.SetText(text) })
			}
			update()
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					update()
				}
			}
		}(stop, tz)
	}

	doSearch := func() {
		city := strings.TrimSpace(cityEdit.Text())
		if city == "" {
			statusLabel.SetText("Please enter a city name.")
			return
		}
		statusLabel.SetText("Searching...")
		locLabel.SetText("")
		coordsLabel.SetText("")
		currentLabel.SetText("")

		go func() {
			loc, err := geocode(city)
			if err != nil {
				mw.Synchronize(func() {
					statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				})
				return
			}
			weather, err := fetchWeather(loc.Lat, loc.Lon)
			if err != nil {
				mw.Synchronize(func() {
					statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				})
				return
			}
			tz, err := time.LoadLocation(loc.Timezone)
			if err != nil {
				tz = time.UTC
			}
			mw.Synchronize(func() {
				locLabel.SetText(fmt.Sprintf("%s, %s", loc.Name, loc.Country))
				coordsLabel.SetText(fmt.Sprintf(
					"Latitude %s, Longitude %s",
					formatLat(loc.Lat), formatLon(loc.Lon)))
				currentLabel.SetText(fmt.Sprintf(
					"%.1f °F   %s   Wind %.1f mph",
					weather.Current.Temperature2m,
					wmoDescription(weather.Current.WeatherCode),
					weather.Current.WindSpeed10m))
				model.Load(weather, tz)
				statusLabel.SetText("")
			})
			startClock(tz)
		}()
	}

	if _, err := (MainWindow{
		AssignTo: &mw,
		Title:    "Travel Weather",
		Size:     Size{Width: 646, Height: 400},
		Layout:   VBox{},
		Font: Font{
			Family:    "Segoe UI",
			PointSize: 10,
		},
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					LineEdit{
						AssignTo:  &cityEdit,
						CueBanner: "Enter city (e.g. Tokyo)",
						OnKeyDown: func(key walk.Key) {
							if key == walk.KeyReturn {
								doSearch()
							}
						},
					},
					PushButton{
						Text:      "Search",
						MinSize:   Size{Width: 90},
						OnClicked: doSearch,
					},
				},
			},
			Label{
				AssignTo:      &locLabel,
				TextAlignment: AlignCenter,
				Font: Font{
					Family:    "Segoe UI Semibold",
					PointSize: 16,
				},
			},
			Label{
				AssignTo:      &coordsLabel,
				TextAlignment: AlignCenter,
				Font: Font{
					Family:    "Segoe UI",
					PointSize: 9,
				},
			},
			Label{
				AssignTo:      &currentLabel,
				TextAlignment: AlignCenter,
				Font: Font{
					Family:    "Segoe UI",
					PointSize: 12,
				},
			},
			Label{
				AssignTo:      &clockLabel,
				TextAlignment: AlignCenter,
				Font: Font{
					Family:    "Cascadia Mono",
					PointSize: 18,
				},
			},
			Label{
				AssignTo:      &statusLabel,
				Text:          "Enter a city and press Search",
				TextAlignment: AlignCenter,
				Font: Font{
					Family:    "Segoe UI",
					PointSize: 9,
					Italic:    true,
				},
			},
			TableView{
				AlternatingRowBG:    true,
				ColumnsOrderable:    false,
				HeaderHidden:        true,
				LastColumnStretched: true,
				Columns: []TableViewColumn{
					{Title: "Day", Width: 60},
					{Title: "Date", Width: 55},
					{Title: "Condition", Width: 140},
					{Title: "High", Width: 80},
					{Title: "Low", Width: 80},
					{Title: "Precip", Width: 80},
					{Title: "Wind", Width: 110},
				},
				Model:         model,
				StretchFactor: 1,
			},
		},
	}.Run()); err != nil {
		log.Fatal(err)
	}
}
