CREATE OR REPLACE FUNCTION public.remove_default_category_seeds(p_tenant_schema TEXT)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    IF p_tenant_schema !~ '^tenant_[0-9a-f_]{32,36}$' THEN
        RAISE EXCEPTION 'invalid tenant schema: %', p_tenant_schema;
    END IF;

    EXECUTE format('
        UPDATE %I.categories c
        SET is_deleted = true,
            deleted_at = now(),
            updated_at = now()
        WHERE c.is_default = true
          AND c.is_deleted = false
          AND (c.name, c.category_type) IN (
              (''Gaji'', ''INCOME''),
              (''Bisnis'', ''INCOME''),
              (''Investasi'', ''INCOME''),
              (''Bonus'', ''INCOME''),
              (''Makanan'', ''EXPENSE''),
              (''Transportasi'', ''EXPENSE''),
              (''Belanja'', ''EXPENSE''),
              (''Kesehatan'', ''EXPENSE''),
              (''Tagihan'', ''EXPENSE''),
              (''Hiburan'', ''EXPENSE''),
              (''Pendidikan'', ''EXPENSE''),
              (''Tabungan'', ''BOTH''),
              (''Transfer'', ''BOTH''),
              (''Lainnya'', ''BOTH'')
          )
          AND NOT EXISTS (
              SELECT 1
              FROM %I.transactions t
              WHERE t.category_id = c.id
                AND t.is_deleted = false
          )', p_tenant_schema, p_tenant_schema);
END;
$$;

GRANT EXECUTE ON FUNCTION public.remove_default_category_seeds(TEXT) TO kasku_finance_svc;

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT schema_name
        FROM information_schema.schemata
        WHERE schema_name ~ '^tenant_[0-9a-f_]{32,36}$'
    LOOP
        PERFORM public.remove_default_category_seeds(r.schema_name);
    END LOOP;
END $$;
