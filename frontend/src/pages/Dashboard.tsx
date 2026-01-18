import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  getUserInfo,
  getAppConfig,
  startMiAuth,
  disconnectMisskey,
  disconnectTwitter,
  generateHeaderToken,
  disableHeaderToken,
  regenerateAPIURLToken,
  logout,
} from '../api'

function Dashboard() {
  const { t, i18n } = useTranslation()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  // Get initial messages from URL params
  const getInitialMessages = () => {
    const success = searchParams.get('success')
    const error = searchParams.get('error')
    let successMsg = ''
    let errorMsg = ''

    if (success) {
      switch (success) {
        case 'misskey_connected':
          successMsg = t('messages.misskeyConnected')
          break
        case 'twitter_connected':
          successMsg = t('messages.twitterConnected')
          break
        default:
          successMsg = t('messages.operationSuccess')
      }
    }

    if (error) {
      errorMsg = `${t('messages.error')}: ${error.replace(/_/g, ' ')}`
    }

    return { successMsg, errorMsg }
  }

  const initialMessages = getInitialMessages()
  const [misskeyInstance, setMisskeyInstance] = useState('')
  const [successMessage, setSuccessMessage] = useState(initialMessages.successMsg)
  const [errorMessage, setErrorMessage] = useState(initialMessages.errorMsg)
  const [generatedToken, setGeneratedToken] = useState<string | null>(null)

  const { data: userInfo, isLoading } = useQuery({
    queryKey: ['userInfo'],
    queryFn: getUserInfo,
  })

  const { data: appConfig, isLoading: isConfigLoading } = useQuery({
    queryKey: ['appConfig'],
    queryFn: getAppConfig,
  })

  // Clear URL params after reading them
  useEffect(() => {
    const success = searchParams.get('success')
    const error = searchParams.get('error')
    if (success || error) {
      navigate('/dashboard', { replace: true })
    }
  }, [searchParams, navigate])

  const miAuthMutation = useMutation({
    mutationFn: startMiAuth,
    onSuccess: (data) => {
      window.location.href = data.auth_url
    },
    onError: (error: Error) => {
      setErrorMessage(error.message)
    },
  })

  const disconnectMisskeyMutation = useMutation({
    mutationFn: disconnectMisskey,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['userInfo'] })
      queryClient.invalidateQueries({ queryKey: ['appConfig'] })
      setSuccessMessage(t('messages.misskeyDisconnected'))
    },
  })

  const disconnectTwitterMutation = useMutation({
    mutationFn: disconnectTwitter,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['userInfo'] })
      setSuccessMessage(t('messages.twitterDisconnected'))
    },
  })

  const generateTokenMutation = useMutation({
    mutationFn: generateHeaderToken,
    onSuccess: (data) => {
      setGeneratedToken(data.token)
      queryClient.invalidateQueries({ queryKey: ['userInfo'] })
    },
  })

  const disableTokenMutation = useMutation({
    mutationFn: disableHeaderToken,
    onSuccess: () => {
      setGeneratedToken(null)
      queryClient.invalidateQueries({ queryKey: ['userInfo'] })
      setSuccessMessage(t('messages.headerTokenDisabled'))
    },
  })

  const regenerateURLMutation = useMutation({
    mutationFn: regenerateAPIURLToken,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['userInfo'] })
      setSuccessMessage(t('messages.apiUrlTokenRegenerated'))
    },
  })

  const logoutMutation = useMutation({
    mutationFn: logout,
    onSuccess: () => {
      queryClient.clear()
      window.location.href = '/login'
    },
  })

  const handleMisskeyConnect = (e: React.FormEvent) => {
    e.preventDefault()
    if (!misskeyInstance.trim()) {
      setErrorMessage(t('messages.enterMisskeyInstance'))
      return
    }
    miAuthMutation.mutate(misskeyInstance)
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    setSuccessMessage(t('messages.copiedToClipboard'))
    setTimeout(() => setSuccessMessage(''), 2000)
  }

  if (isLoading || isConfigLoading) {
    return (
      <div className="flex-center">
        <div className="loading">
          <div className="spinner"></div>
        </div>
      </div>
    )
  }

  if (!userInfo) {
    return <div>{t('messages.errorLoadingUserInfo')}</div>
  }

  const baseUrl = window.location.origin
  const apiPostUrl = `${baseUrl}/api/post/${userInfo.api_url_token}`

  return (
    <>
      <header>
        <h1>
          <span className="spotify-logo">♪</span> Spotify NowPlaying
        </h1>
        <nav>
          <div className="lang-switcher">
            <button 
              className={i18n.language === 'ja' ? 'active' : ''} 
              onClick={() => i18n.changeLanguage('ja')}
            >
              日本語
            </button>
            <button 
              className={i18n.language === 'en' ? 'active' : ''} 
              onClick={() => i18n.changeLanguage('en')}
            >
              EN
            </button>
          </div>
          <button className="secondary" onClick={() => logoutMutation.mutate()}>
            {t('app.logout')}
          </button>
        </nav>
      </header>

      <div className="container">
        {successMessage && (
          <div className="alert success">
            {successMessage}
            <button onClick={() => setSuccessMessage('')} style={{ float: 'right', background: 'none', border: 'none', color: 'inherit' }}>×</button>
          </div>
        )}

        {errorMessage && (
          <div className="alert error">
            {errorMessage}
            <button onClick={() => setErrorMessage('')} style={{ float: 'right', background: 'none', border: 'none', color: 'inherit' }}>×</button>
          </div>
        )}

        {/* Spotify Status */}
        <div className="card">
          <h2>🎵 {t('spotify.title')}</h2>
          <div className="card-row">
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
              {userInfo.spotify_image_url && (
                <img 
                  src={userInfo.spotify_image_url} 
                  alt="Spotify profile" 
                  style={{ 
                    width: '48px', 
                    height: '48px', 
                    borderRadius: '50%',
                    objectFit: 'cover'
                  }} 
                />
              )}
              <span>
                {userInfo.spotify_display_name ? (
                  <>{t('spotify.connectedAs')}: <strong>{userInfo.spotify_display_name}</strong></>
                ) : (
                  <>{t('spotify.connectedAs')}: <strong>{userInfo.spotify_user_id}</strong></>
                )}
              </span>
            </div>
            <span className="status-badge connected">{t('spotify.connected')}</span>
          </div>
        </div>

        {/* Misskey Integration */}
        <div className="card">
          <h2>📝 {t('misskey.title')}</h2>
          {userInfo.misskey_connected ? (
            <>
              <div className="card-row">
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
                  {userInfo.misskey_avatar_url && (
                    <img 
                      src={userInfo.misskey_avatar_url} 
                      alt="Misskey avatar" 
                      style={{ 
                        width: '48px', 
                        height: '48px', 
                        borderRadius: '50%',
                        objectFit: 'cover'
                      }} 
                    />
                  )}
                  <div>
                    <div>
                      <strong>{userInfo.misskey_username || t('misskey.unknown')}</strong>
                      {userInfo.misskey_host && <span style={{ color: '#888' }}>@{userInfo.misskey_host}</span>}
                    </div>
                    {userInfo.misskey_instance_url && (
                      <div style={{ fontSize: '0.9em', color: '#666' }}>{userInfo.misskey_instance_url}</div>
                    )}
                  </div>
                </div>
                <span className="status-badge connected">{t('misskey.connected')}</span>
              </div>
              <button
                className="danger"
                onClick={() => disconnectMisskeyMutation.mutate()}
                disabled={disconnectMisskeyMutation.isPending}
              >
                {t('misskey.disconnect')}
              </button>
            </>
          ) : (
            <>
              <div className="card-row">
                <span>{t('misskey.notConnected')}</span>
                <span className="status-badge disconnected">{t('misskey.disconnected')}</span>
              </div>
              <form onSubmit={handleMisskeyConnect}>
                <div className="input-group">
                  <input
                    type="text"
                    placeholder="misskey.io"
                    value={misskeyInstance}
                    onChange={(e) => setMisskeyInstance(e.target.value)}
                  />
                  <button type="submit" className="primary" disabled={miAuthMutation.isPending}>
                    {miAuthMutation.isPending ? t('misskey.connecting') : t('misskey.connect')}
                  </button>
                </div>
              </form>
            </>
          )}
        </div>

        {/* Twitter Integration */}
        <div className="card">
          <h2>🐦 {t('twitter.title')}</h2>
          {!appConfig?.twitter_available ? (
            <div className="card-row">
              <span style={{ color: '#888' }}>{t('twitter.notAvailable')}</span>
            </div>
          ) : userInfo.twitter_connected ? (
            <>
              <div className="card-row">
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
                  {userInfo.twitter_avatar_url && (
                    <img 
                      src={userInfo.twitter_avatar_url} 
                      alt="Twitter avatar" 
                      style={{ 
                        width: '48px', 
                        height: '48px', 
                        borderRadius: '50%',
                        objectFit: 'cover'
                      }} 
                    />
                  )}
                  <span>
                    {userInfo.twitter_username ? (
                      <>{t('twitter.connectedAs')}: <strong>@{userInfo.twitter_username}</strong></>
                    ) : (
                      <>{t('twitter.connected')}</>
                    )}
                  </span>
                </div>
                <span className="status-badge connected">{t('twitter.connected')}</span>
              </div>
              <button
                className="danger"
                onClick={() => disconnectTwitterMutation.mutate()}
                disabled={disconnectTwitterMutation.isPending}
              >
                {t('twitter.disconnect')}
              </button>
            </>
          ) : appConfig?.twitter_eligibility?.eligible ? (
            <>
              <div className="card-row">
                <span>{t('twitter.notConnected')}</span>
                <span className="status-badge disconnected">{t('twitter.disconnected')}</span>
              </div>
              <a href="/api/twitter/start">
                <button className="primary">{t('twitter.connect')}</button>
              </a>
            </>
          ) : (
            <div className="card-row">
              <span style={{ color: '#888' }}>
                {appConfig?.twitter_eligibility?.reason || t('twitter.connectionNotAvailable')}
              </span>
            </div>
          )}
        </div>

        {/* API URL */}
        <div className="card">
          <h2>🔗 {t('api.urlToken.title')}</h2>
          <p style={{ color: '#888', marginBottom: '1rem', fontSize: '0.9rem' }}>
            {t('api.urlToken.description')}
          </p>
          <div className="code-block">
            <code>{apiPostUrl}</code>
            <button className="copy-button" onClick={() => copyToClipboard(apiPostUrl)}>
              {t('api.urlToken.copy')}
            </button>
          </div>
          <p style={{ color: '#888', marginTop: '0.5rem', fontSize: '0.85rem' }}>
            {t('api.urlToken.optional')}
          </p>
          <div style={{ marginTop: '1rem' }}>
            <button
              className="secondary"
              onClick={() => regenerateURLMutation.mutate()}
              disabled={regenerateURLMutation.isPending}
            >
              {t('api.urlToken.regenerate')}
            </button>
          </div>
        </div>

        {/* Header Token */}
        <div className="card">
          <h2>🔐 {t('api.headerToken.title')}</h2>
          <p style={{ color: '#888', marginBottom: '1rem', fontSize: '0.9rem' }}>
            {t('api.headerToken.description')}
          </p>
          
          {generatedToken && (
            <div className="token-display">
              <p>⚠️ {t('api.headerToken.warning')}</p>
              <code>{generatedToken}</code>
              <button className="copy-button" onClick={() => copyToClipboard(generatedToken)}>
                {t('api.headerToken.copyToken')}
              </button>
            </div>
          )}

          <div className="card-row">
            <span>{t('api.headerToken.status')}: {userInfo.api_header_token_enabled ? t('api.headerToken.enabled') : t('api.headerToken.disabled')}</span>
            <span className={`status-badge ${userInfo.api_header_token_enabled ? 'connected' : 'disconnected'}`}>
              {userInfo.api_header_token_enabled ? t('api.headerToken.enabled') : t('api.headerToken.disabled')}
            </span>
          </div>

          <div style={{ display: 'flex', gap: '0.5rem', marginTop: '1rem' }}>
            <button
              className="primary"
              onClick={() => generateTokenMutation.mutate()}
              disabled={generateTokenMutation.isPending}
            >
              {userInfo.api_header_token_enabled ? t('api.headerToken.regenerate') : t('api.headerToken.generate')}
            </button>
            {userInfo.api_header_token_enabled && (
              <button
                className="danger"
                onClick={() => disableTokenMutation.mutate()}
                disabled={disableTokenMutation.isPending}
              >
                {t('api.headerToken.disable')}
              </button>
            )}
          </div>
        </div>

        {/* Usage Examples */}
        <div className="card">
          <h2>📖 {t('api.examples.title')}</h2>
          <p style={{ color: '#888', marginBottom: '1rem', fontSize: '0.9rem' }}>
            {t('api.examples.description')}
          </p>
          
          <h3 style={{ fontSize: '1rem', marginBottom: '0.5rem' }}>{t('api.examples.basicTitle')}:</h3>
          <div className="code-block">
            <code>curl "{apiPostUrl}"</code>
          </div>

          {userInfo.api_header_token_enabled && (
            <>
              <h3 style={{ fontSize: '1rem', marginTop: '1rem', marginBottom: '0.5rem' }}>{t('api.examples.withHeaderTitle')}:</h3>
              <div className="code-block">
                <code>curl -H "Authorization: Bearer YOUR_TOKEN" "{apiPostUrl}"</code>
              </div>
            </>
          )}

          <h3 style={{ fontSize: '1rem', marginTop: '1rem', marginBottom: '0.5rem' }}>{t('api.examples.misskeyOnlyTitle')}:</h3>
          <div className="code-block">
            <code>curl "{apiPostUrl}?target=misskey"</code>
          </div>

          <h3 style={{ fontSize: '1rem', marginTop: '1rem', marginBottom: '0.5rem' }}>{t('api.examples.twitterOnlyTitle')}:</h3>
          <div className="code-block">
            <code>curl "{apiPostUrl}?target=twitter"</code>
          </div>
        </div>
      </div>
    </>
  )
}

export default Dashboard
