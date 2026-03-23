# dags/weather_pipeline.py

import os
from airflow import DAG
from airflow.operators.bash import BashOperator
from airflow.operators.python import PythonOperator
from datetime import datetime, timedelta
import logging

DB_HOST = os.environ.get("DB_HOST", "postgres")
DB_PORT = os.environ.get("DB_PORT", "5432")
DB_USER = os.environ.get("DB_USER", "warehouse_user")
DB_PASSWORD = os.environ.get("DB_PASSWORD", "warehouse_pass")
DB_NAME = os.environ.get("DB_NAME", "golang_warehouse")
LOCATIONS = os.environ.get("LOCATIONS", "Nairobi,Lagos")

# Default task arguments 
default_args = {
    "owner": "data_team",
    "retries": 2,
    "retry_delay": timedelta(minutes=5),
    "email_on_failure": False,
    "depends_on_past": False,
}

# DAG definition 
with DAG(
    dag_id="weather_etl_pipeline",
    description="Fetch weather data and load into the warehouse",
    default_args=default_args,
    start_date=datetime(2024, 1, 1),
    schedule_interval="0 6 * * *",
    catchup=False,
    tags=["weather", "etl", "warehouse"],
) as dag:

    # Task 1: verify database is reachable
    check_db = BashOperator(
        task_id="check_postgres_connection",
        bash_command="pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER && echo 'DB ready'",
        env={
            "DB_HOST": DB_HOST,
            "DB_PORT": DB_PORT,
            "DB_USER": DB_USER,
        },
    )

    # Task 2: run the Go ETL binary
    run_etl = BashOperator(
        task_id="run_go_etl",
        bash_command="cd /opt/golang_warehouse_1 && go run ./cmd/etl/ 2>&1",
        env={
            "DB_HOST":     DB_HOST,
            "DB_PORT":     DB_PORT,
            "DB_USER":     DB_USER,
            "DB_PASSWORD": DB_PASSWORD,
            "DB_NAME":     DB_NAME,
            "LOCATIONS":   LOCATIONS,
        },
    )

    # Task 3: verify rows were loaded
    def verify_load(**context):
        import psycopg2
        conn = psycopg2.connect(
            host=DB_HOST,
            port=int(DB_PORT),
            user=DB_USER,
            password=DB_PASSWORD,
            dbname=DB_NAME,
        )
        cursor = conn.cursor()
        cursor.execute("""
            SELECT COUNT(*) FROM stg_weather_hourly
            WHERE loaded_at > NOW() - INTERVAL '2 hours'
        """)
        count = cursor.fetchone()[0]
        conn.close()

        logging.info(f"Rows loaded in last 2 hours: {count}")
        if count == 0:
            raise ValueError("No rows loaded — check ETL logs.")
        return count

    verify_data = PythonOperator(
        task_id="verify_data_loaded",
        python_callable=verify_load,
        provide_context=True,
    )

    # Task order
    check_db >> run_etl >> verify_data