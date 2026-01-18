const API_BASE = '/api'

export interface AuthCheckResponse {
  authenticated: boolean
  user_id?: string
  spotify_user_id?: string
}

export interface UserInfo {
  id: string
  spotify_user_id: string
  spotify_display_name?: string
  spotify_image_url?: string
  misskey_connected: boolean
  misskey_instance_url?: string
  misskey_user_id?: string
  misskey_username?: string
  misskey_avatar_url?: string
  misskey_host?: string
  twitter_connected: boolean
  twitter_user_id?: string
  twitter_username?: string
  twitter_avatar_url?: string
  api_url_token: string
  api_header_token_enabled: boolean
}

export interface MiAuthStartResponse {
  auth_url: string
}

export interface HeaderTokenResponse {
  token: string
  message: string
}

export interface RegenerateAPIURLResponse {
  api_url_token: string
}

export async function checkAuth(): Promise<AuthCheckResponse> {
  const res = await fetch(`${API_BASE}/auth/check`)
  if (!res.ok) {
    throw new Error('Failed to check auth')
  }
  return res.json()
}

export async function getUserInfo(): Promise<UserInfo> {
  const res = await fetch(`${API_BASE}/me`)
  if (!res.ok) {
    throw new Error('Failed to get user info')
  }
  return res.json()
}

export async function startMiAuth(instanceUrl: string): Promise<MiAuthStartResponse> {
  const res = await fetch(`${API_BASE}/miauth/start`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ instance_url: instanceUrl }),
  })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Failed to start MiAuth')
  }
  return res.json()
}

export async function disconnectMisskey(): Promise<void> {
  const res = await fetch(`${API_BASE}/miauth`, {
    method: 'DELETE',
  })
  if (!res.ok) {
    throw new Error('Failed to disconnect Misskey')
  }
}

export async function disconnectTwitter(): Promise<void> {
  const res = await fetch(`${API_BASE}/twitter`, {
    method: 'DELETE',
  })
  if (!res.ok) {
    throw new Error('Failed to disconnect Twitter')
  }
}

export async function generateHeaderToken(): Promise<HeaderTokenResponse> {
  const res = await fetch(`${API_BASE}/settings/header-token`, {
    method: 'POST',
  })
  if (!res.ok) {
    throw new Error('Failed to generate header token')
  }
  return res.json()
}

export async function disableHeaderToken(): Promise<void> {
  const res = await fetch(`${API_BASE}/settings/header-token`, {
    method: 'DELETE',
  })
  if (!res.ok) {
    throw new Error('Failed to disable header token')
  }
}

export async function regenerateAPIURLToken(): Promise<RegenerateAPIURLResponse> {
  const res = await fetch(`${API_BASE}/settings/api-url-token/regenerate`, {
    method: 'POST',
  })
  if (!res.ok) {
    throw new Error('Failed to regenerate API URL token')
  }
  return res.json()
}

export async function logout(): Promise<void> {
  const res = await fetch(`${API_BASE}/logout`, {
    method: 'POST',
  })
  if (!res.ok) {
    throw new Error('Failed to logout')
  }
}

// Twitter eligibility
export interface TwitterEligibility {
  eligible: boolean
  reason?: string
}

export interface AppConfig {
  twitter_available: boolean
  twitter_eligibility: TwitterEligibility
}

export async function getAppConfig(): Promise<AppConfig> {
  const res = await fetch(`${API_BASE}/config`)
  if (!res.ok) {
    throw new Error('Failed to get app config')
  }
  return res.json()
}
