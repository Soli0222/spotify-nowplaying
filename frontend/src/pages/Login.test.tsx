import { describe, it, expect } from 'vitest'
import { render, screen } from '../test/test-utils'
import Login from './Login'

describe('Login', () => {
  it('renders the Spotify NowPlaying title', () => {
    render(<Login />)

    expect(screen.getByRole('heading', { level: 1 })).toBeInTheDocument()
    expect(screen.getByText('Spotify NowPlaying')).toBeInTheDocument()
  })

  it('renders the Spotify logo', () => {
    render(<Login />)

    expect(screen.getByText('♪')).toBeInTheDocument()
  })

  it('renders the description text', () => {
    render(<Login />)

    // The translation key is returned as-is due to mocking
    expect(screen.getByText('app.description')).toBeInTheDocument()
  })

  it('renders the login button', () => {
    render(<Login />)

    const button = screen.getByRole('button', { name: 'app.loginWithSpotify' })
    expect(button).toBeInTheDocument()
    expect(button).toHaveClass('primary')
  })

  it('login button links to Spotify auth', () => {
    render(<Login />)

    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', '/api/auth/spotify')
  })
})
