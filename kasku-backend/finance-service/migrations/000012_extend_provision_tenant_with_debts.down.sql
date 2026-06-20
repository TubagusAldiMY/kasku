-- Rollback: tidak bisa drop tabel dari provision_tenant() karena function di-replace
-- dan tenant schemas sudah ada data. Cukup kembalikan function ke versi sebelumnya.
-- Jalankan 000011 down lalu up kembali jika diperlukan full rollback.
-- Untuk menghapus data di tenant yang sudah ada, jalankan manual:
-- DO $$ DECLARE r RECORD; BEGIN FOR r IN SELECT schema_name FROM information_schema.schemata
-- WHERE schema_name ~ '^tenant_[0-9a-f_]+$' LOOP
-- EXECUTE format('DROP TABLE IF EXISTS %I.debt_payments, %I.debts', r.schema_name, r.schema_name);
-- END LOOP; END $$;
SELECT 1; -- no-op placeholder untuk golang-migrate
