--Medallion architecture
--raw
CREATE TABLE IF NOT EXISTS raw_weather (
    id              BIGSERIAL PRIMARY KEY,   
    location        VARCHAR(100) NOT NULL,   
    latitude        DECIMAL(9,6) NOT NULL,   
    longitude       DECIMAL(9,6) NOT NULL,   
    raw_json        JSONB NOT NULL,          
    fetched_at      TIMESTAMPTZ DEFAULT NOW() 
);

--staging: clean and structured
CREATE TABLE IF NOT EXISTS stg_weather_hourly (
    id              BIGSERIAL PRIMARY KEY,
    location        VARCHAR(100) NOT NULL,
    recorded_at     TIMESTAMPTZ NOT NULL,    
    temperature_c   DECIMAL(5,2),           
    windspeed_kmh   DECIMAL(6,2),            
    precipitation_mm DECIMAL(6,2),           
    weathercode     INTEGER,             
    loaded_at       TIMESTAMPTZ DEFAULT NOW(),
    -- Prevent duplicate rows: same location + same hour = skip
    UNIQUE(location, recorded_at)
);

--warehouse layer: aggregated summaries
CREATE TABLE IF NOT EXISTS wh_weather_daily_summary (
    id              BIGSERIAL PRIMARY KEY,
    location        VARCHAR(100) NOT NULL,
    date            DATE NOT NULL,
    avg_temp_c      DECIMAL(5,2),   
    max_temp_c      DECIMAL(5,2),   
    min_temp_c      DECIMAL(5,2),   
    total_precip_mm DECIMAL(6,2),   
    avg_windspeed   DECIMAL(6,2),
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(location, date)
);

-- Indexes to speed up queries
CREATE INDEX IF NOT EXISTS idx_stg_weather_location ON stg_weather_hourly(location);
CREATE INDEX IF NOT EXISTS idx_stg_weather_recorded ON stg_weather_hourly(recorded_at);