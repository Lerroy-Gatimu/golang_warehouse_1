package extractor

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type WeatherResponse struct {
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
    Timezone  string  `json:"timezone"`
    Hourly    struct {
        Time          []string  `json:"time"`
        Temperature   []float64 `json:"temperature_2m"`
        Windspeed     []float64 `json:"windspeed_10m"`
        Precipitation []float64 `json:"precipitation"`
        Weathercode   []int     `json:"weathercode"`
    } `json:"hourly"`
}


var LocationCoords = map[string][2]float64{
    "Nairobi":   {-1.2921, 36.8219},
    "Lagos":     {6.5244, 3.3792},
    "London":    {51.5074, -0.1278},
    "New York":  {40.7128, -74.0060},
    "Tokyo":     {35.6762, 139.6503},
}

func FetchWeather(location, startDate, endDate string) (*WeatherResponse, []byte, error) {
    coords, ok := LocationCoords[location]
    if !ok {
        return nil, nil, fmt.Errorf("unknown location: %s", location)
    }

    lat := coords[0]
    lon := coords[1]

    url := fmt.Sprintf(
        "https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f"+
            "&hourly=temperature_2m,windspeed_10m,precipitation,weathercode"+
            "&start_date=%s&end_date=%s&timezone=auto",
        lat, lon, startDate, endDate,
    )


    client := &http.Client{Timeout: 30 * time.Second}

    resp, err := client.Get(url)
    if err != nil {
        return nil, nil, fmt.Errorf("HTTP GET failed: %w", err)
    }
    defer resp.Body.Close() 

    if resp.StatusCode != http.StatusOK {
        return nil, nil, fmt.Errorf("API returned status %d", resp.StatusCode)
    }


    rawBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, nil, fmt.Errorf("reading response body: %w", err)
    }


    var weather WeatherResponse
    if err := json.Unmarshal(rawBytes, &weather); err != nil {
        return nil, nil, fmt.Errorf("JSON decode failed: %w", err)
    }

    return &weather, rawBytes, nil
}