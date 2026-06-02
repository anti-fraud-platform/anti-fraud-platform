import { Link } from 'react-router-dom'

// Second page - used to demonstrate that routing works.
function About() {
  return (
    <div style={{ padding: '2rem', fontFamily: 'sans-serif' }}>
      <nav style={{ marginBottom: '1.5rem' }}>
        <Link to="/" style={{ marginRight: '1rem' }}>Dashboard</Link>
        <Link to="/about">About</Link>
      </nav>

      <h1>About</h1>
      <p>Real-Time AdTech Anti-Fraud Engine - a system that protects ad budgets from bots.</p>
      <p>This is the frontend part. Basic routing works: you are on the /about page.</p>
    </div>
  )
}

export default About