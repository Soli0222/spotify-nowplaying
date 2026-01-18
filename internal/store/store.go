package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Soli0222/spotify-nowplaying/internal/crypto"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// User represents a user in the database
type User struct {
	ID                    uuid.UUID
	SpotifyUserID         string
	SpotifyAccessToken    sql.NullString
	SpotifyRefreshToken   sql.NullString
	SpotifyTokenExpiresAt sql.NullTime
	MisskeyInstanceURL    sql.NullString
	MisskeyAccessToken    sql.NullString
	MisskeyUserID         sql.NullString
	MisskeyUsername       sql.NullString
	MisskeyAvatarURL      sql.NullString
	MisskeyHost           sql.NullString
	TwitterAccessToken    sql.NullString
	TwitterRefreshToken   sql.NullString
	TwitterTokenExpiresAt sql.NullTime
	TwitterUserID         sql.NullString
	TwitterUsername       sql.NullString
	TwitterAvatarURL      sql.NullString
	APIURLToken           uuid.UUID
	APIHeaderTokenHash    sql.NullString
	APIHeaderTokenEnabled bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// MiAuthSession represents a MiAuth session
type MiAuthSession struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	SessionID   uuid.UUID
	InstanceURL string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

// TwitterPKCESession represents a Twitter PKCE session
type TwitterPKCESession struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	State        string
	CodeVerifier string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// Store provides database operations
type Store struct {
	db *sql.DB
}

// New creates a new Store
func New(databaseURL string) (*Store, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// decryptUserTokens decrypts all encrypted tokens in a User struct
func decryptUserTokens(user *User) error {
	if user == nil {
		return nil
	}

	// Decrypt Spotify tokens
	if user.SpotifyAccessToken.Valid {
		decrypted, err := crypto.DecryptToken(user.SpotifyAccessToken.String)
		if err != nil {
			return fmt.Errorf("failed to decrypt spotify access token: %w", err)
		}
		user.SpotifyAccessToken.String = decrypted
	}
	if user.SpotifyRefreshToken.Valid {
		decrypted, err := crypto.DecryptToken(user.SpotifyRefreshToken.String)
		if err != nil {
			return fmt.Errorf("failed to decrypt spotify refresh token: %w", err)
		}
		user.SpotifyRefreshToken.String = decrypted
	}

	// Decrypt Misskey token
	if user.MisskeyAccessToken.Valid {
		decrypted, err := crypto.DecryptToken(user.MisskeyAccessToken.String)
		if err != nil {
			return fmt.Errorf("failed to decrypt misskey access token: %w", err)
		}
		user.MisskeyAccessToken.String = decrypted
	}

	// Decrypt Twitter tokens
	if user.TwitterAccessToken.Valid {
		decrypted, err := crypto.DecryptToken(user.TwitterAccessToken.String)
		if err != nil {
			return fmt.Errorf("failed to decrypt twitter access token: %w", err)
		}
		user.TwitterAccessToken.String = decrypted
	}
	if user.TwitterRefreshToken.Valid {
		decrypted, err := crypto.DecryptToken(user.TwitterRefreshToken.String)
		if err != nil {
			return fmt.Errorf("failed to decrypt twitter refresh token: %w", err)
		}
		user.TwitterRefreshToken.String = decrypted
	}

	return nil
}

// CreateUser creates a new user
func (s *Store) CreateUser(ctx context.Context, spotifyUserID, accessToken, refreshToken string, expiresAt time.Time) (*User, error) {
	// Encrypt tokens before storing
	encAccessToken, err := crypto.EncryptToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt access token: %w", err)
	}
	encRefreshToken, err := crypto.EncryptToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	user := &User{}
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO users (spotify_user_id, spotify_access_token, spotify_refresh_token, spotify_token_expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (spotify_user_id) DO UPDATE SET
			spotify_access_token = EXCLUDED.spotify_access_token,
			spotify_refresh_token = EXCLUDED.spotify_refresh_token,
			spotify_token_expires_at = EXCLUDED.spotify_token_expires_at,
			updated_at = NOW()
		RETURNING id, spotify_user_id, spotify_access_token, spotify_refresh_token, spotify_token_expires_at,
			misskey_instance_url, misskey_access_token, twitter_access_token, twitter_refresh_token,
			twitter_token_expires_at, api_url_token, api_header_token_hash, api_header_token_enabled,
			created_at, updated_at
	`, spotifyUserID, encAccessToken, encRefreshToken, expiresAt).Scan(
		&user.ID, &user.SpotifyUserID, &user.SpotifyAccessToken, &user.SpotifyRefreshToken,
		&user.SpotifyTokenExpiresAt, &user.MisskeyInstanceURL, &user.MisskeyAccessToken,
		&user.TwitterAccessToken, &user.TwitterRefreshToken, &user.TwitterTokenExpiresAt,
		&user.APIURLToken, &user.APIHeaderTokenHash, &user.APIHeaderTokenEnabled,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Decrypt tokens for the returned user
	if err := decryptUserTokens(user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user := &User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, spotify_user_id, spotify_access_token, spotify_refresh_token, spotify_token_expires_at,
			misskey_instance_url, misskey_access_token, misskey_user_id, misskey_username,
			misskey_avatar_url, misskey_host, twitter_access_token, twitter_refresh_token,
			twitter_token_expires_at, twitter_user_id, twitter_username, twitter_avatar_url,
			api_url_token, api_header_token_hash, api_header_token_enabled,
			created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&user.ID, &user.SpotifyUserID, &user.SpotifyAccessToken, &user.SpotifyRefreshToken,
		&user.SpotifyTokenExpiresAt, &user.MisskeyInstanceURL, &user.MisskeyAccessToken,
		&user.MisskeyUserID, &user.MisskeyUsername, &user.MisskeyAvatarURL, &user.MisskeyHost,
		&user.TwitterAccessToken, &user.TwitterRefreshToken, &user.TwitterTokenExpiresAt,
		&user.TwitterUserID, &user.TwitterUsername, &user.TwitterAvatarURL,
		&user.APIURLToken, &user.APIHeaderTokenHash, &user.APIHeaderTokenEnabled,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := decryptUserTokens(user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserBySpotifyID retrieves a user by Spotify user ID
func (s *Store) GetUserBySpotifyID(ctx context.Context, spotifyUserID string) (*User, error) {
	user := &User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, spotify_user_id, spotify_access_token, spotify_refresh_token, spotify_token_expires_at,
			misskey_instance_url, misskey_access_token, misskey_user_id, misskey_username,
			misskey_avatar_url, misskey_host, twitter_access_token, twitter_refresh_token,
			twitter_token_expires_at, twitter_user_id, twitter_username, twitter_avatar_url,
			api_url_token, api_header_token_hash, api_header_token_enabled,
			created_at, updated_at
		FROM users WHERE spotify_user_id = $1
	`, spotifyUserID).Scan(
		&user.ID, &user.SpotifyUserID, &user.SpotifyAccessToken, &user.SpotifyRefreshToken,
		&user.SpotifyTokenExpiresAt, &user.MisskeyInstanceURL, &user.MisskeyAccessToken,
		&user.MisskeyUserID, &user.MisskeyUsername, &user.MisskeyAvatarURL, &user.MisskeyHost,
		&user.TwitterAccessToken, &user.TwitterRefreshToken, &user.TwitterTokenExpiresAt,
		&user.TwitterUserID, &user.TwitterUsername, &user.TwitterAvatarURL,
		&user.APIURLToken, &user.APIHeaderTokenHash, &user.APIHeaderTokenEnabled,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := decryptUserTokens(user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByAPIToken retrieves a user by API URL token
func (s *Store) GetUserByAPIToken(ctx context.Context, apiToken uuid.UUID) (*User, error) {
	user := &User{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, spotify_user_id, spotify_access_token, spotify_refresh_token, spotify_token_expires_at,
			misskey_instance_url, misskey_access_token, misskey_user_id, misskey_username,
			misskey_avatar_url, misskey_host, twitter_access_token, twitter_refresh_token,
			twitter_token_expires_at, twitter_user_id, twitter_username, twitter_avatar_url,
			api_url_token, api_header_token_hash, api_header_token_enabled,
			created_at, updated_at
		FROM users WHERE api_url_token = $1
	`, apiToken).Scan(
		&user.ID, &user.SpotifyUserID, &user.SpotifyAccessToken, &user.SpotifyRefreshToken,
		&user.SpotifyTokenExpiresAt, &user.MisskeyInstanceURL, &user.MisskeyAccessToken,
		&user.MisskeyUserID, &user.MisskeyUsername, &user.MisskeyAvatarURL, &user.MisskeyHost,
		&user.TwitterAccessToken, &user.TwitterRefreshToken, &user.TwitterTokenExpiresAt,
		&user.TwitterUserID, &user.TwitterUsername, &user.TwitterAvatarURL,
		&user.APIURLToken, &user.APIHeaderTokenHash, &user.APIHeaderTokenEnabled,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := decryptUserTokens(user); err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateMisskeyToken updates the Misskey token and user information for a user
func (s *Store) UpdateMisskeyToken(ctx context.Context, userID uuid.UUID, instanceURL, accessToken, misskeyUserID, username, avatarURL, host string) error {
	encAccessToken, err := crypto.EncryptToken(accessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt misskey token: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE users SET
			misskey_instance_url = $2,
			misskey_access_token = $3,
			misskey_user_id = $4,
			misskey_username = $5,
			misskey_avatar_url = $6,
			misskey_host = $7,
			updated_at = NOW()
		WHERE id = $1
	`, userID, instanceURL, encAccessToken, misskeyUserID, username, avatarURL, host)
	if err != nil {
		return fmt.Errorf("failed to update misskey token: %w", err)
	}
	return nil
}

// UpdateTwitterToken updates the Twitter token and user information for a user
func (s *Store) UpdateTwitterToken(ctx context.Context, userID uuid.UUID, accessToken, refreshToken string, expiresAt time.Time, twitterUserID, username, avatarURL string) error {
	encAccessToken, err := crypto.EncryptToken(accessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt twitter access token: %w", err)
	}
	encRefreshToken, err := crypto.EncryptToken(refreshToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt twitter refresh token: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE users SET
			twitter_access_token = $2,
			twitter_refresh_token = $3,
			twitter_token_expires_at = $4,
			twitter_user_id = $5,
			twitter_username = $6,
			twitter_avatar_url = $7,
			updated_at = NOW()
		WHERE id = $1
	`, userID, encAccessToken, encRefreshToken, expiresAt, twitterUserID, username, avatarURL)
	if err != nil {
		return fmt.Errorf("failed to update twitter token: %w", err)
	}
	return nil
}

// UpdateSpotifyToken updates the Spotify token for a user
func (s *Store) UpdateSpotifyToken(ctx context.Context, userID uuid.UUID, accessToken, refreshToken string, expiresAt time.Time) error {
	encAccessToken, err := crypto.EncryptToken(accessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt spotify access token: %w", err)
	}
	encRefreshToken, err := crypto.EncryptToken(refreshToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt spotify refresh token: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE users SET
			spotify_access_token = $2,
			spotify_refresh_token = $3,
			spotify_token_expires_at = $4,
			updated_at = NOW()
		WHERE id = $1
	`, userID, encAccessToken, encRefreshToken, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to update spotify token: %w", err)
	}
	return nil
}

// RegenerateAPIURLToken regenerates the API URL token for a user
func (s *Store) RegenerateAPIURLToken(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var newToken uuid.UUID
	err := s.db.QueryRowContext(ctx, `
		UPDATE users SET
			api_url_token = gen_random_uuid(),
			updated_at = NOW()
		WHERE id = $1
		RETURNING api_url_token
	`, userID).Scan(&newToken)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to regenerate api url token: %w", err)
	}
	return newToken, nil
}

// SetAPIHeaderToken sets the API header token hash
func (s *Store) SetAPIHeaderToken(ctx context.Context, userID uuid.UUID, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET
			api_header_token_hash = $2,
			api_header_token_enabled = TRUE,
			updated_at = NOW()
		WHERE id = $1
	`, userID, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to set api header token: %w", err)
	}
	return nil
}

// DisableAPIHeaderToken disables the API header token
func (s *Store) DisableAPIHeaderToken(ctx context.Context, userID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET
			api_header_token_hash = NULL,
			api_header_token_enabled = FALSE,
			updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to disable api header token: %w", err)
	}
	return nil
}

// DisconnectMisskey disconnects Misskey from the user account
func (s *Store) DisconnectMisskey(ctx context.Context, userID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET
			misskey_instance_url = NULL,
			misskey_access_token = NULL,
			updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to disconnect misskey: %w", err)
	}
	return nil
}

// DisconnectTwitter disconnects Twitter from the user account
func (s *Store) DisconnectTwitter(ctx context.Context, userID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET
			twitter_access_token = NULL,
			twitter_refresh_token = NULL,
			twitter_token_expires_at = NULL,
			updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to disconnect twitter: %w", err)
	}
	return nil
}

// MiAuth Session operations

// CreateMiAuthSession creates a new MiAuth session
func (s *Store) CreateMiAuthSession(ctx context.Context, userID, sessionID uuid.UUID, instanceURL string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO miauth_sessions (user_id, session_id, instance_url)
		VALUES ($1, $2, $3)
	`, userID, sessionID, instanceURL)
	if err != nil {
		return fmt.Errorf("failed to create miauth session: %w", err)
	}
	return nil
}

// GetMiAuthSession retrieves a MiAuth session by session ID
func (s *Store) GetMiAuthSession(ctx context.Context, sessionID uuid.UUID) (*MiAuthSession, error) {
	session := &MiAuthSession{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, session_id, instance_url, created_at, expires_at
		FROM miauth_sessions WHERE session_id = $1 AND expires_at > NOW()
	`, sessionID).Scan(
		&session.ID, &session.UserID, &session.SessionID, &session.InstanceURL,
		&session.CreatedAt, &session.ExpiresAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get miauth session: %w", err)
	}
	return session, nil
}

// DeleteMiAuthSession deletes a MiAuth session
func (s *Store) DeleteMiAuthSession(ctx context.Context, sessionID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM miauth_sessions WHERE session_id = $1
	`, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete miauth session: %w", err)
	}
	return nil
}

// Twitter PKCE Session operations

// CreateTwitterPKCESession creates a new Twitter PKCE session
func (s *Store) CreateTwitterPKCESession(ctx context.Context, userID uuid.UUID, state, codeVerifier string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO twitter_pkce_sessions (user_id, state, code_verifier)
		VALUES ($1, $2, $3)
	`, userID, state, codeVerifier)
	if err != nil {
		return fmt.Errorf("failed to create twitter pkce session: %w", err)
	}
	return nil
}

// GetTwitterPKCESession retrieves a Twitter PKCE session by state
func (s *Store) GetTwitterPKCESession(ctx context.Context, state string) (*TwitterPKCESession, error) {
	session := &TwitterPKCESession{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, state, code_verifier, created_at, expires_at
		FROM twitter_pkce_sessions WHERE state = $1 AND expires_at > NOW()
	`, state).Scan(
		&session.ID, &session.UserID, &session.State, &session.CodeVerifier,
		&session.CreatedAt, &session.ExpiresAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get twitter pkce session: %w", err)
	}
	return session, nil
}

// DeleteTwitterPKCESession deletes a Twitter PKCE session
func (s *Store) DeleteTwitterPKCESession(ctx context.Context, state string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM twitter_pkce_sessions WHERE state = $1
	`, state)
	if err != nil {
		return fmt.Errorf("failed to delete twitter pkce session: %w", err)
	}
	return nil
}

// CleanupExpiredSessions removes expired sessions
func (s *Store) CleanupExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM miauth_sessions WHERE expires_at < NOW()`)
	if err != nil {
		return fmt.Errorf("failed to cleanup miauth sessions: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM twitter_pkce_sessions WHERE expires_at < NOW()`)
	if err != nil {
		return fmt.Errorf("failed to cleanup twitter pkce sessions: %w", err)
	}
	return nil
}
