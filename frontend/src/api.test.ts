import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  checkAuth,
  getUserInfo,
  startMiAuth,
  disconnectMisskey,
  disconnectTwitter,
  generateHeaderToken,
  disableHeaderToken,
  regenerateAPIURLToken,
  logout,
  getAppConfig,
} from './api'

// Mock fetch globally
const mockFetch = vi.fn()
globalThis.fetch = mockFetch

beforeEach(() => {
  mockFetch.mockReset()
})

describe('checkAuth', () => {
  it('returns authenticated status when successful', async () => {
    const mockResponse = {
      authenticated: true,
      user_id: 'user-123',
      spotify_user_id: 'spotify-456',
    }

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockResponse),
    })

    const result = await checkAuth()

    expect(mockFetch).toHaveBeenCalledWith('/api/auth/check')
    expect(result).toEqual(mockResponse)
  })

  it('throws error when request fails', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
    })

    await expect(checkAuth()).rejects.toThrow('Failed to check auth')
  })
})

describe('getUserInfo', () => {
  it('returns user info when successful', async () => {
    const mockUserInfo = {
      id: 'user-123',
      spotify_user_id: 'spotify-456',
      misskey_connected: true,
      twitter_connected: false,
      api_url_token: 'token-abc',
      api_header_token_enabled: false,
    }

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockUserInfo),
    })

    const result = await getUserInfo()

    expect(mockFetch).toHaveBeenCalledWith('/api/me')
    expect(result).toEqual(mockUserInfo)
  })

  it('throws error when request fails', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
    })

    await expect(getUserInfo()).rejects.toThrow('Failed to get user info')
  })
})

describe('startMiAuth', () => {
  it('returns auth URL when successful', async () => {
    const mockResponse = {
      auth_url: 'https://misskey.tld/miauth/xxx',
    }

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockResponse),
    })

    const result = await startMiAuth('misskey.tld')

    expect(mockFetch).toHaveBeenCalledWith('/api/miauth/start', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ instance_url: 'misskey.tld' }),
    })
    expect(result).toEqual(mockResponse)
  })

  it('throws error with message from API', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      json: () => Promise.resolve({ error: 'Invalid instance' }),
    })

    await expect(startMiAuth('invalid')).rejects.toThrow('Invalid instance')
  })

  it('throws generic error when no error message', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      json: () => Promise.resolve({}),
    })

    await expect(startMiAuth('invalid')).rejects.toThrow('Failed to start MiAuth')
  })
})

describe('disconnectMisskey', () => {
  it('successfully disconnects Misskey', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
    })

    await disconnectMisskey()

    expect(mockFetch).toHaveBeenCalledWith('/api/miauth', {
      method: 'DELETE',
    })
  })

  it('throws error when request fails', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
    })

    await expect(disconnectMisskey()).rejects.toThrow('Failed to disconnect Misskey')
  })
})

describe('disconnectTwitter', () => {
  it('successfully disconnects Twitter', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
    })

    await disconnectTwitter()

    expect(mockFetch).toHaveBeenCalledWith('/api/twitter', {
      method: 'DELETE',
    })
  })

  it('throws error when request fails', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
    })

    await expect(disconnectTwitter()).rejects.toThrow('Failed to disconnect Twitter')
  })
})

describe('generateHeaderToken', () => {
  it('returns token when successful', async () => {
    const mockResponse = {
      token: 'generated-token-xyz',
      message: 'Token generated successfully',
    }

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockResponse),
    })

    const result = await generateHeaderToken()

    expect(mockFetch).toHaveBeenCalledWith('/api/settings/header-token', {
      method: 'POST',
    })
    expect(result).toEqual(mockResponse)
  })

  it('throws error when request fails', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
    })

    await expect(generateHeaderToken()).rejects.toThrow('Failed to generate header token')
  })
})

describe('disableHeaderToken', () => {
  it('successfully disables header token', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
    })

    await disableHeaderToken()

    expect(mockFetch).toHaveBeenCalledWith('/api/settings/header-token', {
      method: 'DELETE',
    })
  })

  it('throws error when request fails', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
    })

    await expect(disableHeaderToken()).rejects.toThrow('Failed to disable header token')
  })
})

describe('regenerateAPIURLToken', () => {
  it('returns new token when successful', async () => {
    const mockResponse = {
      api_url_token: 'new-token-123',
    }

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockResponse),
    })

    const result = await regenerateAPIURLToken()

    expect(mockFetch).toHaveBeenCalledWith('/api/settings/api-url-token/regenerate', {
      method: 'POST',
    })
    expect(result).toEqual(mockResponse)
  })

  it('throws error when request fails', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
    })

    await expect(regenerateAPIURLToken()).rejects.toThrow('Failed to regenerate API URL token')
  })
})

describe('logout', () => {
  it('successfully logs out', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
    })

    await logout()

    expect(mockFetch).toHaveBeenCalledWith('/api/logout', {
      method: 'POST',
    })
  })

  it('throws error when request fails', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
    })

    await expect(logout()).rejects.toThrow('Failed to logout')
  })
})

describe('getAppConfig', () => {
  it('returns app config when successful', async () => {
    const mockConfig = {
      twitter_available: true,
      twitter_eligibility: {
        eligible: true,
      },
    }

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockConfig),
    })

    const result = await getAppConfig()

    expect(mockFetch).toHaveBeenCalledWith('/api/config')
    expect(result).toEqual(mockConfig)
  })

  it('throws error when request fails', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
    })

    await expect(getAppConfig()).rejects.toThrow('Failed to get app config')
  })
})
