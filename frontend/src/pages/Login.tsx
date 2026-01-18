import { useTranslation } from 'react-i18next'

function Login() {
  const { t } = useTranslation()
  
  return (
    <div className="flex-center">
      <div className="login-container">
        <h1>
          <span className="spotify-logo">♪</span> Spotify NowPlaying
        </h1>
        <p>{t('app.description')}</p>
        <a href="/api/auth/spotify">
          <button className="primary">{t('app.loginWithSpotify')}</button>
        </a>
      </div>
    </div>
  )
}

export default Login
