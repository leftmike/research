package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// ---- data types -------------------------------------------------------

type geocodingResult struct {
	Name     string  `json:"name"`
	Country  string  `json:"country"`
	Lat      float64 `json:"latitude"`
	Lon      float64 `json:"longitude"`
	Timezone string  `json:"timezone"`
}

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

type forecastDay struct {
	Date      string
	Icon      string
	Condition string
	High      string
	Low       string
	Precip    string
}

// ---- WMO helpers ------------------------------------------------------

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

func wmoIcon(code int) string {
	switch {
	case code == 0:
		return "[sun]"
	case code <= 3:
		return "[cloud/sun]"
	case code <= 49:
		return "[fog]"
	case code <= 59:
		return "[drizzle]"
	case code <= 69:
		return "[rain]"
	case code <= 79:
		return "[snow]"
	case code <= 84:
		return "[showers]"
	case code <= 86:
		return "[snow shwrs]"
	case code <= 99:
		return "[storm]"
	default:
		return "[?]"
	}
}

// ---- network ----------------------------------------------------------

func geocode(city string) (*geocodingResult, error) {
	u := fmt.Sprintf(
		"https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1&language=en&format=json",
		url.QueryEscape(city))
	resp, err := http.Get(u) //nolint:gosec
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
	resp, err := http.Get(u) //nolint:gosec
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

// ---- app state --------------------------------------------------------

type appState struct {
	th *material.Theme

	// input widgets
	cityEditor widget.Editor
	searchBtn  widget.Clickable

	// display state (guarded by mu)
	mu           sync.Mutex
	locationText string
	clockText    string
	statusText   string
	forecast     []forecastDay

	// clock goroutine control
	clockStop chan struct{}
	tz        *time.Location

	// scroll state for forecast list
	list widget.List
}

func newAppState() *appState {
	s := &appState{
		th:         material.NewTheme(),
		statusText: "Enter a city and press Search.",
	}
	s.cityEditor.SingleLine = true
	s.cityEditor.Submit = true
	s.list.Axis = layout.Vertical
	return s
}

func (s *appState) startClock(w *app.Window, timezone string) {
	if s.clockStop != nil {
		close(s.clockStop)
	}
	tz, err := time.LoadLocation(timezone)
	if err != nil {
		tz = time.UTC
	}
	s.tz = tz
	s.clockStop = make(chan struct{})
	go func(stop chan struct{}) {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case t := <-ticker.C:
				text := t.In(tz).Format("Monday, 02 Jan 2006   15:04:05 MST")
				s.mu.Lock()
				s.clockText = text
				s.mu.Unlock()
				w.Invalidate()
			}
		}
	}(s.clockStop)
}

func (s *appState) doSearch(w *app.Window) {
	city := strings.TrimSpace(s.cityEditor.Text())
	if city == "" {
		s.mu.Lock()
		s.statusText = "Please enter a city name."
		s.mu.Unlock()
		w.Invalidate()
		return
	}

	s.mu.Lock()
	s.statusText = "Searching…"
	s.locationText = ""
	s.clockText = ""
	s.forecast = nil
	s.mu.Unlock()
	w.Invalidate()

	go func() {
		loc, err := geocode(city)
		if err != nil {
			s.mu.Lock()
			s.statusText = "Error: " + err.Error()
			s.mu.Unlock()
			w.Invalidate()
			return
		}

		wx, err := fetchWeather(loc.Lat, loc.Lon)
		if err != nil {
			s.mu.Lock()
			s.statusText = "Error: " + err.Error()
			s.mu.Unlock()
			w.Invalidate()
			return
		}

		days := buildForecast(wx)

		s.mu.Lock()
		s.locationText = fmt.Sprintf("%s, %s  (%.2f°, %.2f°)", loc.Name, loc.Country, loc.Lat, loc.Lon)
		s.statusText = fmt.Sprintf("7-day forecast for %s", loc.Name)
		s.forecast = days
		s.mu.Unlock()

		s.startClock(w, loc.Timezone)
		w.Invalidate()
	}()
}

func buildForecast(wx *weatherResponse) []forecastDay {
	n := len(wx.Daily.Time)
	days := make([]forecastDay, n)
	for i := 0; i < n; i++ {
		code := wx.Daily.WeatherCode[i]
		days[i] = forecastDay{
			Date:      wx.Daily.Time[i],
			Icon:      wmoIcon(code),
			Condition: wmoDescription(code),
			High:      fmt.Sprintf("%.1f °C", wx.Daily.TempMax[i]),
			Low:       fmt.Sprintf("%.1f °C", wx.Daily.TempMin[i]),
		}
		if i < len(wx.Daily.PrecipProb) {
			days[i].Precip = fmt.Sprintf("%d%%", wx.Daily.PrecipProb[i])
		}
	}
	return days
}

// ---- colours ----------------------------------------------------------

var (
	colBg      = color.NRGBA{R: 0xf5, G: 0xf5, B: 0xf5, A: 0xff}
	colCard    = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	colSep     = color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}
	colHeading = color.NRGBA{R: 0x1a, G: 0x23, B: 0x7e, A: 0xff} // indigo 900
	colEven    = color.NRGBA{R: 0xe8, G: 0xea, B: 0xf6, A: 0xff} // light indigo
)

// ---- drawing helpers --------------------------------------------------

func fillRect(gtx layout.Context, c color.NRGBA) {
	paint.FillShape(gtx.Ops, c,
		clip.Rect{Max: gtx.Constraints.Max}.Op())
}

func drawSeparator(gtx layout.Context) layout.Dimensions {
	height := gtx.Dp(1)
	rect := image.Rect(0, 0, gtx.Constraints.Max.X, height)
	paint.FillShape(gtx.Ops, colSep, clip.Rect(rect).Op())
	return layout.Dimensions{Size: image.Point{X: gtx.Constraints.Max.X, Y: height}}
}

// ---- layout -----------------------------------------------------------

func (s *appState) layout(gtx layout.Context, w *app.Window) layout.Dimensions {
	s.mu.Lock()
	locationText := s.locationText
	clockText := s.clockText
	statusText := s.statusText
	forecast := s.forecast
	s.mu.Unlock()

	// check for submit event from the editor
	for {
		ev, ok := s.cityEditor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			s.doSearch(w)
		}
	}
	if s.searchBtn.Clicked(gtx) {
		s.doSearch(w)
	}

	// fill background
	fillRect(gtx, colBg)

	return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceEnd}.Layout(gtx,

			// ---- search row ----
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						ed := material.Editor(s.th, &s.cityEditor, "Enter city (e.g. Tokyo)")
						ed.TextSize = unit.Sp(16)
						return layout.UniformInset(unit.Dp(4)).Layout(gtx, ed.Layout)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(s.th, &s.searchBtn, "Search")
						btn.TextSize = unit.Sp(15)
						return btn.Layout(gtx)
					}),
				)
			}),

			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(drawSeparator),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),

			// ---- location ----
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if locationText == "" {
					return layout.Dimensions{}
				}
				lbl := material.H6(s.th, locationText)
				lbl.Color = colHeading
				lbl.Alignment = 2 // text.Middle
				return layout.UniformInset(unit.Dp(2)).Layout(gtx, lbl.Layout)
			}),

			// ---- clock ----
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if clockText == "" {
					return layout.Dimensions{}
				}
				lbl := material.Body1(s.th, clockText)
				lbl.Color = color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xff}
				lbl.Alignment = 2 // text.Middle
				return layout.UniformInset(unit.Dp(2)).Layout(gtx, lbl.Layout)
			}),

			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),

			// ---- status ----
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body2(s.th, statusText)
				lbl.Color = color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff}
				lbl.Alignment = 2 // text.Middle
				return layout.UniformInset(unit.Dp(2)).Layout(gtx, lbl.Layout)
			}),

			// ---- forecast table ----
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(forecast) == 0 {
					return layout.Dimensions{}
				}
				return s.layoutForecast(gtx, forecast)
			}),
		)
	})
}

func (s *appState) layoutForecast(gtx layout.Context, days []forecastDay) layout.Dimensions {
	rowH := unit.Dp(36)
	colWidths := []unit.Dp{110, 110, 85, 85, 70}
	headers := []string{"Date", "Condition", "High", "Low", "Precip%"}

	cellLayout := func(gtx layout.Context, txt string, bold bool, bg color.NRGBA) layout.Dimensions {
		// draw background
		paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
		var lbl material.LabelStyle
		if bold {
			lbl = material.Body2(s.th, txt)
			lbl.Font.Weight = 700
		} else {
			lbl = material.Body2(s.th, txt)
		}
		return layout.UniformInset(unit.Dp(6)).Layout(gtx, lbl.Layout)
	}

	rowLayout := func(gtx layout.Context, cells []string, bold bool, bg color.NRGBA) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			func() []layout.FlexChild {
				children := make([]layout.FlexChild, len(cells))
				for i, txt := range cells {
					i, txt := i, txt
					w := colWidths[i]
					children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints = layout.Exact(image.Point{
							X: gtx.Dp(w),
							Y: gtx.Dp(rowH),
						})
						return cellLayout(gtx, txt, bold, bg)
					})
				}
				return children
			}()...,
		)
	}

	return s.list.Layout(gtx, len(days)+1, func(gtx layout.Context, idx int) layout.Dimensions {
		if idx == 0 {
			return rowLayout(gtx, headers, true, colHeading)
		}
		d := days[idx-1]
		cells := []string{d.Date, d.Condition, d.High, d.Low, d.Precip}
		bg := colCard
		if idx%2 == 0 {
			bg = colEven
		}
		return rowLayout(gtx, cells, false, bg)
	})
}

// ---- main -------------------------------------------------------------

func main() {
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Travel Weather"),
			app.Size(unit.Dp(600), unit.Dp(520)),
		)

		state := newAppState()
		var ops op.Ops

		for {
			e := w.Event()
			switch e := e.(type) {
			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)
				state.layout(gtx, w)
				e.Frame(gtx.Ops)
			case app.DestroyEvent:
				os.Exit(0)
			}
		}
	}()

	app.Main()
}
