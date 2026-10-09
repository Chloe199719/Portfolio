ALTER TABLE accounts ADD COLUMN totp_confirmed BOOLEAN NOT NULL DEFAULT false;
-- Existing factors remain enrolled: a password alone must never reveal their keys.
UPDATE accounts SET totp_confirmed=true WHERE totp_secret!='';
