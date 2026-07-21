import { useState, useEffect } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { Shield, Fingerprint, Search } from 'lucide-react'
import { register } from '../api/auth'

function Register() {
  const navigate = useNavigate()
  const [form, setForm] = useState({ username: '', password: '' })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState('')

  useEffect(() => {
    const token = localStorage.getItem('token')
    if (token) {
      navigate('/analytics', { replace: true })
    }
  }, [navigate])

  const handleChange = (e) => {
    setForm({ ...form, [e.target.name]: e.target.value })
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setSuccess('')
    setLoading(true)

    try {
      const data = await register(form.username, form.password)
      setSuccess(data.message || 'Registration successful. Redirecting to login...')
      // Redirect to login
      setTimeout(() => {
        navigate('/login', { replace: true })
      }, 1500)
    } catch (err) {
      const message =
        err.response?.data?.error || 'Registration failed. Please try again.'
      setError(message)
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-app-bg p-4">
      <div className="w-full max-w-sm space-y-6 rounded-xl border border-border bg-surface p-6">
        <div className="flex flex-col items-center text-center">
          <div className="text-primary mb-3">
            <Shield size={40}>
              <Fingerprint size={10} x={7} y={7} />
              <Search size={12} x={12} y={12} />
            </Shield>
          </div>
          <h1 className="text-lg font-semibold text-text-main">ANTIFRAUD</h1>
          <p className="text-xs text-text-muted">Click Fraud Protection</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="rounded-lg bg-danger-light px-4 py-2 text-sm text-danger">
              {error}
            </div>
          )}
          {success && (
            <div className="rounded-lg bg-success-light px-4 py-2 text-sm text-success">
              {success}
            </div>
          )}

          <div>
            <label htmlFor="username" className="mb-1 block text-sm font-medium text-text-main">
              Username
            </label>
            <input
              id="username"
              name="username"
              type="text"
              value={form.username}
              onChange={handleChange}
              required
              className="w-full rounded-lg border border-border bg-app-bg px-3 py-2 text-sm text-text-main placeholder:text-text-muted focus:border-primary focus:outline-none"
              placeholder="Choose a username"
            />
          </div>

          <div>
            <label htmlFor="password" className="mb-1 block text-sm font-medium text-text-main">
              Password
            </label>
            <input
              id="password"
              name="password"
              type="password"
              value={form.password}
              onChange={handleChange}
              required
              className="w-full rounded-lg border border-border bg-app-bg px-3 py-2 text-sm text-text-main placeholder:text-text-muted focus:border-primary focus:outline-none"
              placeholder="Choose a password"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {loading ? 'Creating account…' : 'Create account'}
          </button>
        </form>

        <p className="text-center text-xs text-text-muted">
          Already have an account?{' '}
          <Link to="/login" className="text-primary hover:underline">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  )
}

export default Register