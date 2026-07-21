import client from './client'

// POST /v1/auth/login
// Body: { username, password }
// Backend returns: { token }
export async function login(username, password) {
  const response = await client.post('/v1/auth/login', { username, password })
  return response.data
}

// POST /v1/auth/register
// Body: { username, password }
// Backend returns: { message: "user created" }
export async function register(username, password) {
  const response = await client.post('/v1/auth/register', { username, password })
  return response.data
}

// GET /v1/auth/me
// Header: Authorization: Bearer eyJhbGci...
// Backend returns: { id, username, role }
export async function getMe() {
  const response = await client.get('/v1/auth/me')
  return response.data
}