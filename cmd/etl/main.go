package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/joho/godotenv"
    "github.com/Lerroy-Gatimu/golang_warehouse_1/internal/extractor"
    "github.com/Lerroy-Gatimu/golang_warehouse_1/internal/loader"
    "github.com/Lerroy-Gatimu/golang_warehouse_1/internal/transformer"
)

func main() {
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found, reading from environment")
    }

    dbHost := getEnv("DB_HOST", "127.0.0.1")
    dbPort := getEnv("DB_PORT", "5432")
    dbUser := getEnv("DB_USER", "postgres")
    dbPass := getEnv("DB_PASSWORD", "")
    dbName := getEnv("DB_NAME", "golang_warehouse")

    connStr := fmt.Sprintf(
    "postgres://%s:%s@%s:%s/%s?sslmode=disable",
    dbUser, dbPass, dbHost, dbPort, dbName,
)

    // Create a context — contexts control timeouts and cancellation 
    ctx := context.Background()

    db, err := loader.New(ctx, connStr)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close() 

    // Set date range: fetch last 7 days of weather data
    endDate := time.Now().Format("2006-01-02")
    startDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

    log.Printf("Starting ETL pipeline | Date range: %s to %s", startDate, endDate)

    // Define which cities to process
    locations := []string{"Nairobi", "Lagos", "London", "New York", "Tokyo"}

    totalHourly := 0
    totalSummaries := 0

    // Process each city
    for _, location := range locations {
        log.Printf("Processing: %s", location)

        weatherData, rawBytes, err := extractor.FetchWeather(location, startDate, endDate)
        if err != nil {
            log.Printf("❌ Extract failed for %s: %v", location, err)
            continue // Skip this city, move to next
        }
        log.Printf("   Extracted %d hourly records", len(weatherData.Hourly.Time))

        // EXTRACT
        coords := extractor.LocationCoords[location]
        if err := db.SaveRaw(ctx, location, coords[0], coords[1], rawBytes); err != nil {
            log.Printf("   ⚠️  Raw save failed: %v", err)
            // Don't stop — continue with transform/load
        }

        // TRANSFORM
        hourlyRecords, err := transformer.TransformHourly(location, weatherData)
        if err != nil {
            log.Printf("❌ Transform failed for %s: %v", location, err)
            continue
        }

        dailySummaries := transformer.AggregateDailyFromHourly(hourlyRecords)

        // LOAD 
        n, err := db.SaveHourly(ctx, hourlyRecords)
        if err != nil {
            log.Printf("❌ Load (hourly) failed for %s: %v", location, err)
            continue
        }
        totalHourly += n

        m, err := db.SaveDailySummaries(ctx, dailySummaries)
        if err != nil {
            log.Printf("❌ Load (daily) failed for %s: %v", location, err)
            continue
        }
        totalSummaries += m

        log.Printf("   ✅ Inserted %d hourly rows, %d daily summaries", n, m)
    }

    log.Printf("Pipeline complete | %d hourly rows | %d daily summaries", totalHourly, totalSummaries)
}


func getEnv(key, fallback string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return fallback
}