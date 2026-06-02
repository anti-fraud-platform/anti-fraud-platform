import { Link } from 'react-router-dom'

// Вторая страница — нужна, чтобы продемонстрировать работающий роутинг.
function About() {
  return (
    <div style={{ padding: '2rem', fontFamily: 'sans-serif' }}>
      <nav style={{ marginBottom: '1.5rem' }}>
        <Link to="/" style={{ marginRight: '1rem' }}>Dashboard</Link>
        <Link to="/about">About</Link>
      </nav>

      <h1>О проекте</h1>
      <p>Real-Time AdTech Anti-Fraud Engine — система защиты рекламных бюджетов от ботов.</p>
      <p>Это фронтенд-часть. Базовый роутинг работает: ты на странице /about.</p>
    </div>
  )
}

export default About