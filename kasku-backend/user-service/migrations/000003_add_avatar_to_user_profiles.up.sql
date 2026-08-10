-- Foto profil disimpan langsung di Postgres sebagai BYTEA, bukan object storage.
--
-- Alasan: stack ini belum punya MinIO/S3, dan avatar sudah dinormalisasi server-side
-- ke maksimal 512x512 JPEG (~60KB) sebelum ditulis. Pada skala itu satu baris avatar
-- jauh di bawah ambang TOAST-berat, ikut serta dalam backup Postgres yang sudah ada,
-- dan tidak menambah container, kredensial, maupun kebijakan retensi baru.
--
-- ponytail: BYTEA cukup sampai ratusan ribu user. Kalau avatar sudah jutaan atau
-- traffic gambar mulai mendominasi bandwidth DB, pindahkan ke object storage dan
-- ganti kolom ini jadi avatar_url TEXT — pembacaannya sudah terisolasi di
-- UserProfileRepository, jadi perubahannya tidak merembet ke handler.
ALTER TABLE public.user_profiles
    ADD COLUMN IF NOT EXISTS avatar_image      BYTEA       NULL,
    ADD COLUMN IF NOT EXISTS avatar_mime       TEXT        NULL,
    ADD COLUMN IF NOT EXISTS avatar_updated_at TIMESTAMPTZ NULL;

-- Ketiganya harus terisi bersama atau kosong bersama. Tanpa ini, baris dengan
-- gambar tapi tanpa MIME akan lolos ke handler dan menghasilkan response yang
-- Content-Type-nya tidak bisa ditentukan.
ALTER TABLE public.user_profiles
    DROP CONSTRAINT IF EXISTS user_profiles_avatar_complete;
ALTER TABLE public.user_profiles
    ADD CONSTRAINT user_profiles_avatar_complete CHECK (
        (avatar_image IS NULL AND avatar_mime IS NULL AND avatar_updated_at IS NULL)
        OR
        (avatar_image IS NOT NULL AND avatar_mime IS NOT NULL AND avatar_updated_at IS NOT NULL)
    );

-- Batas ukuran ditegakkan di DB juga, bukan hanya di handler: handler bisa diganti
-- atau di-bypass lewat jalur lain, sedangkan constraint ini selalu berlaku.
-- 256KB memberi kelonggaran besar di atas hasil resize (~60KB).
ALTER TABLE public.user_profiles
    DROP CONSTRAINT IF EXISTS user_profiles_avatar_size;
ALTER TABLE public.user_profiles
    ADD CONSTRAINT user_profiles_avatar_size CHECK (
        avatar_image IS NULL OR octet_length(avatar_image) <= 262144
    );
