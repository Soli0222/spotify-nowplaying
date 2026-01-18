import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '../test/test-utils'
import Dashboard from './Dashboard'
import * as api from '../api'

// Mock the API module
vi.mock('../api', () => ({
  getUserInfo: vi.fn(),
  getAppConfig: vi.fn(),
  startMiAuth: vi.fn(),
  disconnectMisskey: vi.fn(),
  disconnectTwitter: vi.fn(),
  generateHeaderToken: vi.fn(),
  disableHeaderToken: vi.fn(),
  regenerateAPIURLToken: vi.fn(),
  logout: vi.fn(),
}))

const mockUserInfo: api.UserInfo = {
  id: 'user-123',
  spotify_user_id: 'spotify-456',
  spotify_display_name: 'Test User',
  spotify_image_url: 'https://example.com/avatar.jpg',
  misskey_connected: false,
  twitter_connected: false,
  api_url_token: 'test-token-abc',
  api_header_token_enabled: false,
}

const mockAppConfig: api.AppConfig = {
  twitter_available: true,
  twitter_eligibility: {
    eligible: true,
  },
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.getUserInfo).mockResolvedValue(mockUserInfo)
  vi.mocked(api.getAppConfig).mockResolvedValue(mockAppConfig)
})

describe('Dashboard', () => {
  it('shows loading spinner initially', () => {
    // Make the API call hang
    vi.mocked(api.getUserInfo).mockImplementation(() => new Promise(() => {}))

    render(<Dashboard />)

    expect(document.querySelector('.spinner')).toBeInTheDocument()
  })

  it('renders dashboard after loading', async () => {
    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByText('Spotify NowPlaying')).toBeInTheDocument()
    })
  })

  it('displays Spotify user info', async () => {
    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByText('Test User')).toBeInTheDocument()
    })
  })

  it('shows Spotify profile image when available', async () => {
    render(<Dashboard />)

    await waitFor(() => {
      const img = screen.getByAltText('Spotify profile')
      expect(img).toHaveAttribute('src', 'https://example.com/avatar.jpg')
    })
  })

  it('shows logout button', async () => {
    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'app.logout' })).toBeInTheDocument()
    })
  })

  it('shows language switcher buttons', async () => {
    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByText('日本語')).toBeInTheDocument()
      expect(screen.getByText('EN')).toBeInTheDocument()
    })
  })
})

describe('Dashboard - Misskey Integration', () => {
  it('shows connect form when Misskey not connected', async () => {
    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByPlaceholderText('misskey.io')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'misskey.connect' })).toBeInTheDocument()
    })
  })

  it('shows disconnect button when Misskey connected', async () => {
    vi.mocked(api.getUserInfo).mockResolvedValue({
      ...mockUserInfo,
      misskey_connected: true,
      misskey_username: 'testuser',
      misskey_instance_url: 'https://misskey.io',
      misskey_avatar_url: 'https://misskey.io/avatar.jpg',
      misskey_host: 'misskey.io',
    })

    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByText('testuser')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'misskey.disconnect' })).toBeInTheDocument()
    })
  })
})

describe('Dashboard - Twitter Integration', () => {
  it('shows connect button when Twitter available and eligible', async () => {
    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'twitter.connect' })).toBeInTheDocument()
    })
  })

  it('shows not available message when Twitter not available', async () => {
    vi.mocked(api.getAppConfig).mockResolvedValue({
      twitter_available: false,
      twitter_eligibility: { eligible: false },
    })

    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByText('twitter.notAvailable')).toBeInTheDocument()
    })
  })

  it('shows disconnect button when Twitter connected', async () => {
    vi.mocked(api.getUserInfo).mockResolvedValue({
      ...mockUserInfo,
      twitter_connected: true,
      twitter_username: 'twitteruser',
      twitter_avatar_url: 'https://twitter.com/avatar.jpg',
    })

    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByText('@twitteruser', { exact: false })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'twitter.disconnect' })).toBeInTheDocument()
    })
  })

  it('shows eligibility reason when not eligible', async () => {
    vi.mocked(api.getAppConfig).mockResolvedValue({
      twitter_available: true,
      twitter_eligibility: {
        eligible: false,
        reason: 'Misskey connection required',
      },
    })

    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByText('Misskey connection required')).toBeInTheDocument()
    })
  })
})

describe('Dashboard - API Settings', () => {
  it('shows API URL with token', async () => {
    render(<Dashboard />)

    await waitFor(() => {
      const codeBlocks = screen.getAllByText(/test-token-abc/)
      expect(codeBlocks.length).toBeGreaterThan(0)
    })
  })

  it('shows copy button for API URL', async () => {
    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'api.urlToken.copy' })).toBeInTheDocument()
    })
  })

  it('shows regenerate button for API URL', async () => {
    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'api.urlToken.regenerate' })).toBeInTheDocument()
    })
  })

  it('shows header token status as disabled by default', async () => {
    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByText(/api.headerToken.status/)).toBeInTheDocument()
    })
  })

  it('shows generate button when header token disabled', async () => {
    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'api.headerToken.generate' })).toBeInTheDocument()
    })
  })

  it('shows disable button when header token enabled', async () => {
    vi.mocked(api.getUserInfo).mockResolvedValue({
      ...mockUserInfo,
      api_header_token_enabled: true,
    })

    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'api.headerToken.disable' })).toBeInTheDocument()
    })
  })
})

describe('Dashboard - Usage Examples', () => {
  it('shows curl examples with token', async () => {
    render(<Dashboard />)

    await waitFor(() => {
      const curlExamples = screen.getAllByText(/curl.*test-token-abc/)
      expect(curlExamples.length).toBe(3) // basic, misskey, twitter
    })
  })

  it('shows target parameter examples', async () => {
    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByText(/target=misskey/)).toBeInTheDocument()
      expect(screen.getByText(/target=twitter/)).toBeInTheDocument()
    })
  })

  it('shows header token example when enabled', async () => {
    vi.mocked(api.getUserInfo).mockResolvedValue({
      ...mockUserInfo,
      api_header_token_enabled: true,
    })

    render(<Dashboard />)

    await waitFor(() => {
      expect(screen.getByText(/Authorization: Bearer YOUR_TOKEN/)).toBeInTheDocument()
    })
  })
})
