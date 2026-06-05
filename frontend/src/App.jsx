import { Routes, Route } from 'react-router-dom'
import Dashboard from './pages/Dashboard'
import About from './pages/About'

// Application routing.
function App() {
  return (
    <Routes>
      <Route path="/" element={<Dashboard />} />
      <Route path="/about" element={<About />} />
    </Routes>
  )
}

export default App