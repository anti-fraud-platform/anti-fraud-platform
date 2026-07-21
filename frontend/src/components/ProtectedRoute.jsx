import { Navigate } from 'react-router-dom'

function ProtectedRoute({ children }) {
  const token = localStorage.getItem('token')

  if (!token || token.length === 0) {
    return <Navigate to="/login" replace />
  }

  return children
}

export default ProtectedRoute