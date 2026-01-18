import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import App from './App'

// Mock the API module
const mockCheckAuth = vi.fn()
const mockGetUserInfo = vi.fn()
const mockGetAppConfig = vi.fn()

vi.mock('./api', () => ({
  checkAuth: () => mockCheckAuth(),
  getUserInfo: () => mockGetUserInfo(),
  getAppConfig: () => mockGetAppConfig(),
}))

// Mock the page components
vi.mock('./pages/Login', () => ({
  default: () => <div data-testid="login-page">Login Page</div>,
}))

vi.mock('./pages/Dashboard', () => ({
  default: () => <div data-testid="dashboard-page">Dashboard Page</div>,
}))

const createTestWrapper = (initialEntries: string[] = ['/']) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  })

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={initialEntries}>{children}</MemoryRouter>
    </QueryClientProvider>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('App', () => {
  it('shows loading spinner while checking auth', () => {
    mockCheckAuth.mockImplementation(() => new Promise(() => {}))

    render(<App />, { wrapper: createTestWrapper() })

    expect(document.querySelector('.spinner')).toBeInTheDocument()
  })

  it('redirects to login when not authenticated', async () => {
    mockCheckAuth.mockResolvedValue({ authenticated: false })

    render(<App />, { wrapper: createTestWrapper(['/']) })

    await waitFor(() => {
      expect(screen.getByTestId('login-page')).toBeInTheDocument()
    })
  })

  it('redirects to dashboard when authenticated', async () => {
    mockCheckAuth.mockResolvedValue({
      authenticated: true,
      user_id: 'user-123',
      spotify_user_id: 'spotify-456',
    })
    mockGetUserInfo.mockResolvedValue({
      id: 'user-123',
      spotify_user_id: 'spotify-456',
      misskey_connected: false,
      twitter_connected: false,
      api_url_token: 'token',
      api_header_token_enabled: false,
    })
    mockGetAppConfig.mockResolvedValue({
      twitter_available: true,
      twitter_eligibility: { eligible: true },
    })

    render(<App />, { wrapper: createTestWrapper(['/']) })

    await waitFor(() => {
      expect(screen.getByTestId('dashboard-page')).toBeInTheDocument()
    })
  })

  it('shows login page at /login when not authenticated', async () => {
    mockCheckAuth.mockResolvedValue({ authenticated: false })

    render(<App />, { wrapper: createTestWrapper(['/login']) })

    await waitFor(() => {
      expect(screen.getByTestId('login-page')).toBeInTheDocument()
    })
  })

  it('redirects from /login to /dashboard when already authenticated', async () => {
    mockCheckAuth.mockResolvedValue({
      authenticated: true,
      user_id: 'user-123',
      spotify_user_id: 'spotify-456',
    })
    mockGetUserInfo.mockResolvedValue({
      id: 'user-123',
      spotify_user_id: 'spotify-456',
      misskey_connected: false,
      twitter_connected: false,
      api_url_token: 'token',
      api_header_token_enabled: false,
    })
    mockGetAppConfig.mockResolvedValue({
      twitter_available: true,
      twitter_eligibility: { eligible: true },
    })

    render(<App />, { wrapper: createTestWrapper(['/login']) })

    await waitFor(() => {
      expect(screen.getByTestId('dashboard-page')).toBeInTheDocument()
    })
  })

  it('redirects from /dashboard to /login when not authenticated', async () => {
    mockCheckAuth.mockResolvedValue({ authenticated: false })

    render(<App />, { wrapper: createTestWrapper(['/dashboard']) })

    await waitFor(() => {
      expect(screen.getByTestId('login-page')).toBeInTheDocument()
    })
  })

  it('handles unknown routes by redirecting appropriately', async () => {
    mockCheckAuth.mockResolvedValue({ authenticated: false })

    render(<App />, { wrapper: createTestWrapper(['/unknown-path']) })

    await waitFor(() => {
      expect(screen.getByTestId('login-page')).toBeInTheDocument()
    })
  })
})
