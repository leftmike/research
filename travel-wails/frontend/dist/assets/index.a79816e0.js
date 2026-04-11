(function(){const e=document.createElement("link").relList;if(e&&e.supports&&e.supports("modulepreload"))return;for(const t of document.querySelectorAll('link[rel="modulepreload"]'))c(t);new MutationObserver(t=>{for(const o of t)if(o.type==="childList")for(const s of o.addedNodes)s.tagName==="LINK"&&s.rel==="modulepreload"&&c(s)}).observe(document,{childList:!0,subtree:!0});function i(t){const o={};return t.integrity&&(o.integrity=t.integrity),t.referrerpolicy&&(o.referrerPolicy=t.referrerpolicy),t.crossorigin==="use-credentials"?o.credentials="include":t.crossorigin==="anonymous"?o.credentials="omit":o.credentials="same-origin",o}function c(t){if(t.ep)return;t.ep=!0;const o=i(t);fetch(t.href,o)}})();function b(n){return window.go.main.App.GetWeather(n)}document.querySelector("#app").innerHTML=`
  <div class="container">
    <h1 class="title">\u2708 Travel Weather</h1>

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
`;const a=document.getElementById("city-input"),r=document.getElementById("search-btn"),l=document.getElementById("status"),u=document.getElementById("info-block"),p=document.getElementById("location-label"),g=document.getElementById("clock-label"),f=document.getElementById("forecast-block"),m=document.getElementById("forecast-body");a.focus();let d=null,h=null;function v(n){d!==null&&clearInterval(d),h=new Intl.DateTimeFormat("en-GB",{timeZone:n,weekday:"long",day:"2-digit",month:"short",year:"numeric",hour:"2-digit",minute:"2-digit",second:"2-digit",hour12:!1,timeZoneName:"short"});const e=()=>{g.textContent=h.format(new Date)};e(),d=setInterval(e,1e3)}async function y(){const n=a.value.trim();if(!n){l.textContent="Please enter a city name.";return}l.textContent="Searching\u2026",u.classList.add("hidden"),f.classList.add("hidden"),r.disabled=!0;try{const e=await b(n),i=e.location;p.textContent=`${i.name}, ${i.country}  (${i.lat.toFixed(2)}\xB0, ${i.lon.toFixed(2)}\xB0)`,v(i.timezone),u.classList.remove("hidden"),m.innerHTML="";for(const c of e.forecast){const t=document.createElement("tr");t.innerHTML=`
              <td>${c.date}</td>
              <td class="icon-cell">${c.icon}</td>
              <td>${c.condition}</td>
              <td class="num">${c.high.toFixed(1)} \xB0C</td>
              <td class="num">${c.low.toFixed(1)} \xB0C</td>
              <td class="num">${c.precip}%</td>
            `,m.appendChild(t)}f.classList.remove("hidden"),l.textContent=`7-day forecast for ${i.name}`}catch(e){l.textContent=`Error: ${e}`}finally{r.disabled=!1}}r.addEventListener("click",y);a.addEventListener("keydown",n=>{n.key==="Enter"&&y()});
