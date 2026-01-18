import { Routes, Route, Navigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import { checkAuth } from './api'

function App() {
  const { data: authData, isLoading } = useQuery({
    queryKey: ['auth'],
    queryFn: checkAuth,
  })

  if (isLoading) {
    return (
      <div className="flex-center">
        <div className="loading">
          <div className="spinner"></div>
        </div>
      </div>
    )
  }

  const isAuthenticated = authData?.authenticated ?? false

  return (
    <Routes>
      <Route
        path="/login"
        element={isAuthenticated ? <Navigate to="/dashboard" replace /> : <Login />}
      />
      <Route
        path="/dashboard"
        element={isAuthenticated ? <Dashboard /> : <Navigate to="/login" replace />}
      />
      <Route
        path="/"
        element={<Navigate to={isAuthenticated ? "/dashboard" : "/login"} replace />}
      />
      {/* Fallback for SPA */}
      <Route
        path="*"
        element={<Navigate to={isAuthenticated ? "/dashboard" : "/login"} replace />}
      />
    </Routes>
  )
}

export default App
