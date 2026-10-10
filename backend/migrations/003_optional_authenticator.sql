-- Confirmed factors remain enabled; unfinished bootstrap enrollment is optional.
ALTER TABLE accounts ADD COLUMN totp_pending_at TIMESTAMPTZ;
