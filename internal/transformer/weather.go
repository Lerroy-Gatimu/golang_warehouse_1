package transformer

import (
    "time"

    "github.com/Lerroy-Gatimu/golang_warehouse_1/internal/extractor"
)

type HourlyRecord struct {
    Location      string
    RecordedAt    time.Time
    TemperatureC  float64
    WindspeedKmh  float64
    PrecipMM      float64
    Weathercode   int
}

type DailySummary struct {
    Location    string
    Date        time.Time
    AvgTempC    float64
    MaxTempC    float64
    MinTempC    float64
    TotalPrecip float64
    AvgWindspeed float64
}

func TransformHourly(location string, w *extractor.WeatherResponse) ([]HourlyRecord, error) {
    records := make([]HourlyRecord, 0, len(w.Hourly.Time))

    for i, timeStr := range w.Hourly.Time {
        t, err := time.Parse("2006-01-02T15:04", timeStr)
        if err != nil {
            continue
        }

        record := HourlyRecord{
            Location:     location,
            RecordedAt:   t,
            TemperatureC: safeFloat(w.Hourly.Temperature, i),
            WindspeedKmh: safeFloat(w.Hourly.Windspeed, i),
            PrecipMM:     safeFloat(w.Hourly.Precipitation, i),
            Weathercode:  safeInt(w.Hourly.Weathercode, i),
        }
        records = append(records, record)
    }

    return records, nil
}


func AggregateDailyFromHourly(records []HourlyRecord) []DailySummary {
    type dayKey struct {
        Location string
        Date     string
    }
    groups := make(map[dayKey][]HourlyRecord)

    for _, r := range records {
        key := dayKey{
            Location: r.Location,
            Date:     r.RecordedAt.Format("2006-01-02"),
        }
        groups[key] = append(groups[key], r)
    }

    summaries := make([]DailySummary, 0, len(groups))

    for key, hrs := range groups {
        date, _ := time.Parse("2006-01-02", key.Date)
        summary := DailySummary{
            Location: key.Location,
            Date:     date,
            MaxTempC: hrs[0].TemperatureC,
            MinTempC: hrs[0].TemperatureC,
        }

        var tempSum, windSum, precipSum float64
        for _, h := range hrs {
            tempSum += h.TemperatureC
            windSum += h.WindspeedKmh
            precipSum += h.PrecipMM
            if h.TemperatureC > summary.MaxTempC {
                summary.MaxTempC = h.TemperatureC
            }
            if h.TemperatureC < summary.MinTempC {
                summary.MinTempC = h.TemperatureC
            }
        }

        n := float64(len(hrs))
        summary.AvgTempC = tempSum / n
        summary.AvgWindspeed = windSum / n
        summary.TotalPrecip = precipSum

        summaries = append(summaries, summary)
    }

    return summaries
}


func safeFloat(slice []float64, i int) float64 {
    if i < len(slice) {
        return slice[i]
    }
    return 0
}

func safeInt(slice []int, i int) int {
    if i < len(slice) {
        return slice[i]
    }
    return 0
}