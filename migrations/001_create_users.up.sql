-- Users table for storing account information
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    spotify_user_id VARCHAR(255) UNIQUE NOT NULL,
    spotify_access_token TEXT,
    spotify_refresh_token TEXT,
    spotify_token_expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Misskey integration (MiAuth)
    misskey_instance_url VARCHAR(512),
    misskey_access_token TEXT,
    
    -- Twitter integration (OAuth 2.0 PKCE)
    twitter_access_token TEXT,
    twitter_refresh_token TEXT,
    twitter_token_expires_at TIMESTAMP WITH TIME ZONE,
    
    -- API posting URL
    api_url_token UUID UNIQUE DEFAULT gen_random_uuid(),
    
    -- Optional header token authentication
    api_header_token_hash VARCHAR(64),  -- SHA-256 hash of the token
    api_header_token_enabled BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_spotify_user_id ON users(spotify_user_id);
CREATE INDEX IF NOT EXISTS idx_users_api_url_token ON users(api_url_token);

-- MiAuth sessions table for tracking ongoing authentications
CREATE TABLE IF NOT EXISTS miauth_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id UUID NOT NULL,
    instance_url VARCHAR(512) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() + INTERVAL '10 minutes'
);

CREATE INDEX IF NOT EXISTS idx_miauth_sessions_session_id ON miauth_sessions(session_id);

-- Twitter PKCE sessions table for tracking ongoing authentications
CREATE TABLE IF NOT EXISTS twitter_pkce_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state VARCHAR(64) NOT NULL,
    code_verifier VARCHAR(128) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() + INTERVAL '10 minutes'
);

CREATE INDEX IF NOT EXISTS idx_twitter_pkce_sessions_state ON twitter_pkce_sessions(state);
