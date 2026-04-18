import './style.css';
import {GetWeather} from '../wailsjs/go/main/App';

// ── DOM scaffold ──────────────────────────────────────────────────────────────
document.querySelector('#app').innerHTML = `
  <div class="container">
    <h1 class="title">✈ Travel Weather</h1>

    <div class="search-row">
      <input id="city-input" class="city-input" type="text"
             placeholder="Enter city (e.g. Tokyo)" autocomplete="off" />
      <button id="search-btn" class="search-btn">Search</button>
    </div>

    <div id="status" class="status">Enter a city and press Search.</div>

    <div id="info-block" class="info-block hidden">
      <div id="location-label" class="location-label"></div>
      <div id="clock-label" class="clock-label"></div>
    </div>

    <div id="forecast-block" class="forecast-block hidden">
      <table class="forecast-table">
        <thead>
          <tr>
            <th>Date</th>
            <th></th>
            <th>Condition</th>
            <th>High</th>
            <th>Low</th>
            <th>Precip</th>
          </tr>
        </thead>
        <tbody id="forecast-body"></tbody>
      </table>
    </div>
  </div>
`;

// ── Elements ──────────────────────────────────────────────────────────────────
const cityInput    = document.getElementById('city-input');
const searchBtn    = document.getElementById('search-btn');
const statusEl     = document.getElementById('status');
const infoBlock    = document.getElementById('info-block');
const locationLabel= document.getElementById('location-label');
const clockLabel   = document.getElementById('clock-label');
const forecastBlock= document.getElementById('forecast-block');
const forecastBody = document.getElementById('forecast-body');

cityInput.focus();

// ── Clock ─────────────────────────────────────────────────────────────────────
let clockTimer = null;
let clockFmt   = null;

function startClock(timezone) {
    if (clockTimer !== null) clearInterval(clockTimer);
    clockFmt = new Intl.DateTimeFormat('en-GB', {
        timeZone:     timezone,
        weekday:      'long',
        day:          '2-digit',
        month:        'short',
        year:         'numeric',
        hour:         '2-digit',
        minute:       '2-digit',
        second:       '2-digit',
        hour12:       false,
        timeZoneName: 'short',
    });
    const tick = () => { clockLabel.textContent = clockFmt.format(new Date()); };
    tick();
    clockTimer = setInterval(tick, 1000);
}

// ── Search ────────────────────────────────────────────────────────────────────
async function doSearch() {
    const city = cityInput.value.trim();
    if (!city) {
        statusEl.textContent = 'Please enter a city name.';
        return;
    }

    statusEl.textContent = 'Searching…';
    infoBlock.classList.add('hidden');
    forecastBlock.classList.add('hidden');
    searchBtn.disabled = true;

    try {
        const result = await GetWeather(city);
        const loc = result.location;

        locationLabel.textContent =
            `${loc.name}, ${loc.country}  (${loc.lat.toFixed(2)}°, ${loc.lon.toFixed(2)}°)`;

        startClock(loc.timezone);
        infoBlock.classList.remove('hidden');

        // Build forecast rows
        forecastBody.innerHTML = '';
        for (const day of result.forecast) {
            const tr = document.createElement('tr');
            tr.innerHTML = `
              <td>${day.date}</td>
              <td class="icon-cell">${day.icon}</td>
              <td>${day.condition}</td>
              <td class="num">${day.high.toFixed(1)} °C</td>
              <td class="num">${day.low.toFixed(1)} °C</td>
              <td class="num">${day.precip}%</td>
            `;
            forecastBody.appendChild(tr);
        }
        forecastBlock.classList.remove('hidden');
        statusEl.textContent = `7-day forecast for ${loc.name}`;
    } catch (err) {
        statusEl.textContent = `Error: ${err}`;
    } finally {
        searchBtn.disabled = false;
    }
}

searchBtn.addEventListener('click', doSearch);
cityInput.addEventListener('keydown', (e) => { if (e.key === 'Enter') doSearch(); });
