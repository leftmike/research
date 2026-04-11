export namespace main {
	
	export class DayForecast {
	    date: string;
	    icon: string;
	    condition: string;
	    high: number;
	    low: number;
	    precip: number;
	
	    static createFrom(source: any = {}) {
	        return new DayForecast(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.icon = source["icon"];
	        this.condition = source["condition"];
	        this.high = source["high"];
	        this.low = source["low"];
	        this.precip = source["precip"];
	    }
	}
	export class Location {
	    name: string;
	    country: string;
	    lat: number;
	    lon: number;
	    timezone: string;
	
	    static createFrom(source: any = {}) {
	        return new Location(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.country = source["country"];
	        this.lat = source["lat"];
	        this.lon = source["lon"];
	        this.timezone = source["timezone"];
	    }
	}
	export class WeatherResult {
	    location: Location;
	    forecast: DayForecast[];
	
	    static createFrom(source: any = {}) {
	        return new WeatherResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.location = this.convertValues(source["location"], Location);
	        this.forecast = this.convertValues(source["forecast"], DayForecast);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

