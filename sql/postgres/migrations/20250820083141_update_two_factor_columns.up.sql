ALTER TABLE owners ADD COLUMN IF NOT EXISTS otp_data TEXT;

UPDATE owners 
SET otp_data = jsonb_build_object(
    'otp_secret', otp_secret,
    'otp_confirmed', COALESCE(otp_confirmed, false)
)::text
WHERE otp_secret IS NOT NULL OR otp_confirmed IS NOT NULL;