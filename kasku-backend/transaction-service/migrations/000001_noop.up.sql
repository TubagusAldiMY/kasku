-- transaction-service tidak membuat schema sendiri.
-- Semua tabel (transactions, categories, financial_accounts) dibuat oleh
-- provision_tenant() di finance-service migrations.
-- File ini diperlukan agar golang-migrate tidak error saat startup.
SELECT 1;
