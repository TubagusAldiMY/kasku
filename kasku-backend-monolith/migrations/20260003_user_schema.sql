-- 20260003: User management schema — user_profiles

CREATE TABLE IF NOT EXISTS user_mgmt.user_profiles (
    user_id      UUID         PRIMARY KEY,
    email        TEXT         NOT NULL,
    username     VARCHAR(50)  NOT NULL UNIQUE,
    display_name VARCHAR(100) NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TRIGGER user_profiles_set_updated_at
    BEFORE UPDATE ON user_mgmt.user_profiles
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at_timestamp();
