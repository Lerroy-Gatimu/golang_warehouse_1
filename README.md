# golang_warehouse_1

A production-style **data warehouse pipeline** built with Go, PostgreSQL, and Apache Airflow.  
Ingests real hourly weather data from the [Open-Meteo API](https://open-meteo.com/), applies a three-layer warehouse architecture, and automates daily runs via Airflow.

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [Tech Stack](#tech-stack)
3. [Architecture](#architecture)
4. [Project Structure](#project-structure)
5. [Prerequisites](#prerequisites)
6. [Setup & Installation](#setup--installation)
7. [Running the Pipeline](#running-the-pipeline)
8. [Airflow Orchestration](#airflow-orchestration)
9. [Database Schema](#database-schema)
10. [Key Concepts Explained](#key-concepts-explained)
11. [Querying Your Warehouse](#querying-your-warehouse)
12. [Troubleshooting](#troubleshooting)

---

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

## Architecture

```
Open-Meteo API
      │
      ▼
┌─────────────┐
│  EXTRACTOR  │  Go: fetches JSON from API
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ TRANSFORMER │  Go: cleans data, builds daily summaries
└──────┬──────┘
       │
       ▼
┌────────────────────────────────┐
│         PostgreSQL             │
│  raw_weather        (raw)      │
│  stg_weather_hourly (staging)  │
│  wh_weather_daily_summary (wh) │
└────────────────────────────────┘
       ▲
       │ orchestrates
┌─────────────┐
│   Airflow   │  Schedules daily runs, monitors health
└─────────────┘
```

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

## Prerequisites

Install the following before proceeding:

| Tool | Version | Install |
|---|---|---|
| Go | 1.22+ | https://go.dev/dl/ |
| Docker Desktop | Latest | https://www.docker.com/products/docker-desktop/ |
| Git | Any | https://git-scm.com/ |

Verify installs:
```bash
go version
docker --version
docker compose version
```

---

## Setup & Installation

### 1. Clone / Create the Project

```bash
mkdir golang_warehouse_1
cd golang_warehouse_1
go mod init github.com/yourusername/golang_warehouse_1
```

### 2. Create All Directories

```bash
mkdir -p cmd/etl internal/extractor internal/transformer internal/loader db/migrations dags
```

### 3. Install Go Dependencies

```bash
go get github.com/jackc/pgx/v5
go get github.com/joho/godotenv
```

### 4. Create the `.env` File

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=warehouse_user
DB_PASSWORD=warehouse_pass
DB_NAME=golang_warehouse
```

### 5. Copy All Source Files

Copy the files provided in the setup guide into their respective paths (see Project Structure above).

### 6. Start Infrastructure

```bash
docker compose up -d
```

This starts PostgreSQL and Airflow in the background. The first run takes a few minutes to download images.

Check containers are running:
```bash
docker compose ps
```

### 7. Verify the Database

```bash
docker exec -it warehouse_postgres psql -U postgres -d golang_warehouse -c "\dt"
```

You should see the three warehouse tables listed.

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

---

## Key Concepts Explained

### What is ETL?
ETL stands for **Extract, Transform, Load** — the three phases of moving data from a source to a destination:
- **Extract**: Pull data from an API or database
- **Transform**: Clean, validate, and reshape it
- **Load**: Write the clean data to your warehouse

### What is a Data Warehouse?
A data warehouse is a database optimized for analysis rather than live transactions. It stores historical data in layers (raw → clean → aggregated) so analysts can run queries efficiently.

### What is Airflow?
Apache Airflow is a platform for defining, scheduling, and monitoring data workflows. You write pipelines as Python code (DAGs), and Airflow handles running them on schedule, retrying failures, and showing you what happened.

### What is a Connection Pool?
Instead of opening and closing a new database connection for every query (slow), a connection pool keeps several connections open and reuses them. The `pgxpool` library handles this automatically.

### What is an Upsert?
An upsert is `INSERT ... ON CONFLICT DO UPDATE/NOTHING`. It inserts a new row if it doesn't exist, or updates/skips it if it does. This makes pipelines **idempotent** — you can re-run them safely without creating duplicate data.

---

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




