#!/bin/bash
set -e

# Runs as kasku_superuser during PostgreSQL first-init.
# kasku_auth is created automatically via POSTGRES_DB env var.
# This script creates the remaining databases and all service users.
#
# NOTE: passwords must not contain single-quotes.

echo "[init-db] Creating additional databases..."

# Connect to POSTGRES_DB (kasku_auth) — always exists at this point.
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-SQL
    CREATE DATABASE kasku_billing;
    CREATE DATABASE kasku_finance;
    CREATE DATABASE kasku_price;
    CREATE DATABASE kasku_admin;
    CREATE DATABASE kasku_user;
    CREATE DATABASE kasku_notification;
SQL

echo "[init-db] Creating service users..."

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-SQL
    CREATE USER kasku_auth_svc         WITH PASSWORD '${KASKU_AUTH_DB_PASS}';
    CREATE USER kasku_billing_svc      WITH PASSWORD '${KASKU_BILLING_DB_PASS}';
    CREATE USER kasku_finance_svc      WITH PASSWORD '${KASKU_FINANCE_DB_PASS}';
    CREATE USER kasku_transaction_svc  WITH PASSWORD '${KASKU_TRANSACTION_DB_PASS}';
    CREATE USER kasku_investment_svc   WITH PASSWORD '${KASKU_INVESTMENT_DB_PASS}';
    CREATE USER kasku_sync_svc         WITH PASSWORD '${KASKU_SYNC_DB_PASS}';
    CREATE USER kasku_user_svc         WITH PASSWORD '${KASKU_USER_DB_PASS}';
    CREATE USER kasku_price_svc        WITH PASSWORD '${KASKU_PRICE_DB_PASS}';
    CREATE USER kasku_notification_svc WITH PASSWORD '${KASKU_NOTIFICATION_DB_PASS}';
    CREATE USER kasku_admin_svc        WITH PASSWORD '${KASKU_ADMIN_DB_PASS}';
    CREATE USER kasku_admin_read       WITH PASSWORD '${KASKU_ADMIN_READ_DB_PASS}';
SQL

echo "[init-db] Enabling required PostgreSQL extensions..."
for db in kasku_auth kasku_billing kasku_finance kasku_price kasku_admin kasku_user kasku_notification; do
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$db" <<-SQL
        CREATE EXTENSION IF NOT EXISTS pgcrypto;
SQL
done

echo "[init-db] Granting kasku_auth access..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "kasku_auth" <<-SQL
    GRANT CONNECT ON DATABASE kasku_auth TO kasku_auth_svc;
    GRANT USAGE   ON SCHEMA  public      TO kasku_auth_svc;
    GRANT CREATE  ON SCHEMA  public      TO kasku_auth_svc;

    GRANT CONNECT ON DATABASE kasku_auth TO kasku_admin_read;
    GRANT USAGE   ON SCHEMA  public      TO kasku_admin_read;
SQL

echo "[init-db] Granting kasku_billing access..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "kasku_billing" <<-SQL
    GRANT CONNECT ON DATABASE kasku_billing TO kasku_billing_svc;
    GRANT USAGE   ON SCHEMA  public         TO kasku_billing_svc;
    GRANT CREATE  ON SCHEMA  public         TO kasku_billing_svc;

    -- user-service masih butuh INSERT/SELECT untuk CreateFreeSubscription
    -- (cross-DB grant terdokumentasi; subscriptions tetap dimiliki billing-service).
    GRANT CONNECT ON DATABASE kasku_billing TO kasku_user_svc;
    GRANT USAGE   ON SCHEMA  public         TO kasku_user_svc;

    GRANT CONNECT ON DATABASE kasku_billing TO kasku_admin_read;
    GRANT USAGE   ON SCHEMA  public         TO kasku_admin_read;
SQL

echo "[init-db] Granting kasku_finance access..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "kasku_finance" <<-SQL
    GRANT CONNECT ON DATABASE kasku_finance TO kasku_finance_svc;
    GRANT CREATE  ON DATABASE kasku_finance TO kasku_finance_svc;
    GRANT CONNECT ON DATABASE kasku_finance TO kasku_transaction_svc;
    GRANT CONNECT ON DATABASE kasku_finance TO kasku_investment_svc;
    GRANT CONNECT ON DATABASE kasku_finance TO kasku_sync_svc;
    GRANT CONNECT ON DATABASE kasku_finance TO kasku_user_svc;

    GRANT USAGE ON SCHEMA public TO kasku_finance_svc;
    GRANT CREATE ON SCHEMA public TO kasku_finance_svc;
    GRANT USAGE ON SCHEMA public TO kasku_transaction_svc;
    GRANT USAGE ON SCHEMA public TO kasku_investment_svc;
    GRANT USAGE ON SCHEMA public TO kasku_sync_svc;
    GRANT USAGE ON SCHEMA public TO kasku_user_svc;
    -- EXECUTE on provision_tenant/deprovision_tenant granted by finance-service migration.
    -- Per-table grants on tenant schemas applied by provision_tenant function at runtime.
SQL

echo "[init-db] Granting kasku_price access..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "kasku_price" <<-SQL
    GRANT CONNECT ON DATABASE kasku_price TO kasku_price_svc;
    GRANT USAGE   ON SCHEMA  public       TO kasku_price_svc;
    GRANT CREATE  ON SCHEMA  public       TO kasku_price_svc;
SQL

echo "[init-db] Granting kasku_admin access..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "kasku_admin" <<-SQL
    GRANT CONNECT ON DATABASE kasku_admin TO kasku_admin_svc;
    GRANT USAGE   ON SCHEMA  public       TO kasku_admin_svc;
    GRANT CREATE  ON SCHEMA  public       TO kasku_admin_svc;
SQL

echo "[init-db] Granting kasku_user access..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "kasku_user" <<-SQL
    GRANT CONNECT ON DATABASE kasku_user TO kasku_user_svc;
    GRANT USAGE   ON SCHEMA  public      TO kasku_user_svc;
    GRANT CREATE  ON SCHEMA  public      TO kasku_user_svc;
SQL

echo "[init-db] Granting kasku_notification access..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "kasku_notification" <<-SQL
    GRANT CONNECT ON DATABASE kasku_notification TO kasku_notification_svc;
    GRANT USAGE   ON SCHEMA  public              TO kasku_notification_svc;
    GRANT CREATE  ON SCHEMA  public              TO kasku_notification_svc;
SQL

echo "[init-db] Database initialization complete."
