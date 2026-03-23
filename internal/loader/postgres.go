package loader

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/Lerroy-Gatimu/golang_warehouse_1/internal/transformer"
)

type DB struct {
    pool *pgxpool.Pool
}

// connString format: "postgres://user:password@host:port/dbname"
func New(ctx context.Context, connString string) (*DB, error) {
    pool, err := pgxpool.New(ctx, connString)
    if err != nil {
        return nil, fmt.Errorf("creating connection pool: %w", err)
    }

    // Test the connection
    if err := pool.Ping(ctx); err != nil {
        return nil, fmt.Errorf("pinging database: %w", err)
    }

    log.Println("✅ Connected to PostgreSQL")
    return &DB{pool: pool}, nil
}


func (db *DB) Close() {
    db.pool.Close()
}

// SaveRaw stores the original API JSON response in the raw layer.
// We always save raw data first — it's our audit trail.
func (db *DB) SaveRaw(ctx context.Context, location string, lat, lon float64, rawJSON []byte) error {
    _, err := db.pool.Exec(ctx, `
        INSERT INTO raw_weather (location, latitude, longitude, raw_json)
        VALUES ($1, $2, $3, $4)
    `, location, lat, lon, rawJSON)
    // Note: $1, $2, $3, $4 are parameterized placeholders — this prevents SQL injection
    return err
}

// SaveHourly bulk-inserts hourly records into the staging table.
// Uses a batch insert for performance — one round-trip to DB for all rows.
func (db *DB) SaveHourly(ctx context.Context, records []transformer.HourlyRecord) (int, error) {
    if len(records) == 0 {
        return 0, nil
    }

    // Use a transaction — either ALL rows insert or NONE do (atomic operation)
    tx, err := db.pool.Begin(ctx)
    if err != nil {
        return 0, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback(ctx) // Rollback if we return early due to error

    inserted := 0
    for _, r := range records {
        tag, err := tx.Exec(ctx, `
            INSERT INTO stg_weather_hourly 
                (location, recorded_at, temperature_c, windspeed_kmh, precipitation_mm, weathercode)
            VALUES ($1, $2, $3, $4, $5, $6)
            ON CONFLICT (location, recorded_at) DO NOTHING
        `, r.Location, r.RecordedAt, r.TemperatureC, r.WindspeedKmh, r.PrecipMM, r.Weathercode)

        if err != nil {
            return 0, fmt.Errorf("inserting hourly record: %w", err)
        }
        inserted += int(tag.RowsAffected())
    }

    if err := tx.Commit(ctx); err != nil {
        return 0, fmt.Errorf("commit transaction: %w", err)
    }

    return inserted, nil
}

func (db *DB) SaveDailySummaries(ctx context.Context, summaries []transformer.DailySummary) (int, error) {
    if len(summaries) == 0 {
        return 0, nil
    }

    tx, err := db.pool.Begin(ctx)
    if err != nil {
        return 0, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    inserted := 0
    for _, s := range summaries {
        tag, err := tx.Exec(ctx, `
            INSERT INTO wh_weather_daily_summary
                (location, date, avg_temp_c, max_temp_c, min_temp_c, total_precip_mm, avg_windspeed)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            ON CONFLICT (location, date) DO UPDATE SET
                avg_temp_c    = EXCLUDED.avg_temp_c,
                max_temp_c    = EXCLUDED.max_temp_c,
                min_temp_c    = EXCLUDED.min_temp_c,
                total_precip_mm = EXCLUDED.total_precip_mm,
                avg_windspeed = EXCLUDED.avg_windspeed
        `, s.Location, s.Date, s.AvgTempC, s.MaxTempC, s.MinTempC, s.TotalPrecip, s.AvgWindspeed)

        if err != nil {
            return 0, fmt.Errorf("inserting daily summary: %w", err)
        }
        inserted += int(tag.RowsAffected())
    }

    if err := tx.Commit(ctx); err != nil {
        return 0, fmt.Errorf("commit: %w", err)
    }

    _ = json.Marshal

    return inserted, nil
}