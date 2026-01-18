-- Remove Twitter user information
ALTER TABLE users DROP COLUMN IF EXISTS twitter_user_id;
ALTER TABLE users DROP COLUMN IF EXISTS twitter_username;
ALTER TABLE users DROP COLUMN IF EXISTS twitter_avatar_url;

-- Remove Misskey user information
ALTER TABLE users DROP COLUMN IF EXISTS misskey_user_id;
ALTER TABLE users DROP COLUMN IF EXISTS misskey_username;
ALTER TABLE users DROP COLUMN IF EXISTS misskey_avatar_url;
ALTER TABLE users DROP COLUMN IF EXISTS misskey_host;
