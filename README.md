# golang_warehouse_1

A production-style **data warehouse pipeline** built with Go, PostgreSQL, and Apache Airflow.  
Ingests real hourly weather data from the [Open-Meteo API](https://open-meteo.com/), applies a three-layer warehouse architecture, and automates daily runs via Airflow.


## Project Overview

`golang_warehouse_1` demonstrates a real-world ETL (Extract, Transform, Load) pipeline that:

- **Extracts** hourly weather data for 5 global cities from the Open-Meteo API
- **Transforms** raw JSON into clean, typed records and daily aggregates
- **Loads** data into a three-layer PostgreSQL warehouse (Raw → Staging → Warehouse)
- **Orchestrates** daily runs on a schedule using Apache Airflow

---

## Tech Stack

| Layer | Technology | Purpose |
|---|---|---|
| ETL Worker | Go 1.22+ | High-performance data pipeline |
| Database | PostgreSQL 16 | Relational data warehouse storage |
| Orchestration | Apache Airflow 2.9 | Pipeline scheduling and monitoring |
| Infrastructure | Docker Compose | Local environment setup |
| Data Source | Open-Meteo API | Free, real-time weather data |

---


### Three-Layer Warehouse Pattern

| Layer | Table | Description |
|---|---|---|
| **Raw** | `raw_weather` | Original API response stored as JSON. Never modified. |
| **Staging** | `stg_weather_hourly` | Cleaned, typed hourly records. Ready for querying. |
| **Warehouse** | `wh_weather_daily_summary` | Aggregated daily summaries optimized for reporting. |

---

## Project Structure

```
golang_warehouse_1/
├── cmd/
│   └── etl/
│       └── main.go                 ← Pipeline entry point
├── internal/
│   ├── extractor/
│   │   └── weather.go              ← Open-Meteo API client
│   ├── transformer/
│   │   └── weather.go              ← Data cleaning and aggregation
│   └── loader/
│       └── postgres.go             ← PostgreSQL write operations
├── db/
│   └── migrations/
│       └── 001_init.sql            ← Table definitions
├── dags/
│   └── weather_pipeline.py         ← Airflow DAG
├── docker-compose.yml              ← Postgres + Airflow services
├── .env                            ← Local configuration (not committed)
├── go.mod                          ← Go module definition
└── README.md                       ← This file
```

---

## Running the Pipeline

### Manual Run (Development)

```bash
go run ./cmd/etl/
```

Expected output:
```
✅ Connected to PostgreSQL
Starting ETL pipeline | Date range: 2024-01-08 to 2024-01-15
Processing: Nairobi
   Extracted 168 hourly records
   ✅ Inserted 168 hourly rows, 7 daily summaries
Processing: Lagos
   ...
Pipeline complete | 840 hourly rows | 35 daily summaries
```

### Build a Binary (Production)

```bash
go build -o bin/etl ./cmd/etl/
./bin/etl
```

---

## Airflow Orchestration

### Access the Airflow UI

Open http://localhost:8080 in your browser.

- Username: `admin`
- Password: `admin`

### Trigger a Manual Run

1. Navigate to **DAGs** in the top menu
2. Find `weather_etl_pipeline`
3. Click the **▶ (Trigger DAG)** button
4. Watch the task states: green = success, red = failed

### Schedule

The DAG runs automatically every day at **6:00 AM** (cron: `0 6 * * *`).

### DAG Tasks

| Task | Description |
|---|---|
| `check_postgres_connection` | Verifies the database is reachable |
| `run_go_etl` | Executes the Go ETL binary |
| `verify_data_loaded` | Confirms rows were inserted |

---

## Database Schema

### `raw_weather` (Raw Layer)

| Column | Type | Description |
|---|---|---|
| id | BIGSERIAL | Primary key |
| location | VARCHAR(100) | City name |
| latitude | DECIMAL(9,6) | GPS latitude |
| longitude | DECIMAL(9,6) | GPS longitude |
| raw_json | JSONB | Original API response |
| fetched_at | TIMESTAMPTZ | Ingestion timestamp |

### `stg_weather_hourly` (Staging Layer)

| Column | Type | Description |
|---|---|---|
| id | BIGSERIAL | Primary key |
| location | VARCHAR(100) | City name |
| recorded_at | TIMESTAMPTZ | Hour this reading covers |
| temperature_c | DECIMAL(5,2) | Temperature in Celsius |
| windspeed_kmh | DECIMAL(6,2) | Wind speed km/h |
| precipitation_mm | DECIMAL(6,2) | Rainfall mm |
| weathercode | INTEGER | WMO weather condition code |

### `wh_weather_daily_summary` (Warehouse Layer)

| Column | Type | Description |
|---|---|---|
| id | BIGSERIAL | Primary key |
| location | VARCHAR(100) | City name |
| date | DATE | Summary date |
| avg_temp_c | DECIMAL(5,2) | Average daily temperature |
| max_temp_c | DECIMAL(5,2) | Hottest point |
| min_temp_c | DECIMAL(5,2) | Coldest point |
| total_precip_mm | DECIMAL(6,2) | Total rainfall |
| avg_windspeed | DECIMAL(6,2) | Average wind speed |


## Querying Your Warehouse

Connect to the database:
```bash
docker exec -it warehouse_postgres psql -U postgres -d golang_warehouse
```

### Useful Queries

```sql
-- How many rows per city?
SELECT location, COUNT(*) FROM stg_weather_hourly GROUP BY location;

-- Hottest day per city in the last 7 days
SELECT location, date, max_temp_c
FROM wh_weather_daily_summary
ORDER BY max_temp_c DESC
LIMIT 10;

-- Average temperature by city
SELECT location, ROUND(AVG(avg_temp_c)::numeric, 2) AS overall_avg
FROM wh_weather_daily_summary
GROUP BY location
ORDER BY overall_avg DESC;

-- Rainiest days
SELECT location, date, total_precip_mm
FROM wh_weather_daily_summary
WHERE total_precip_mm > 5
ORDER BY total_precip_mm DESC;
```

---




