-- Add Twitter user information
ALTER TABLE users ADD COLUMN IF NOT EXISTS twitter_user_id VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS twitter_username VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS twitter_avatar_url TEXT;

-- Add Misskey user information
ALTER TABLE users ADD COLUMN IF NOT EXISTS misskey_user_id VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS misskey_username VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS misskey_avatar_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS misskey_host VARCHAR(512);
