import { Routes, Route } from 'react-router-dom'
import Dashboard from './pages/Dashboard'
import Logs from './pages/Logs'
import Blacklist from './pages/Blacklist'

// Application routing: three admin pages.
function App() {
  return (
    <Routes>
      <Route path="/" element={<Dashboard />} />
      <Route path="/logs" element={<Logs />} />
      <Route path="/blacklist" element={<Blacklist />} />
    </Routes>
  )
}

export default App