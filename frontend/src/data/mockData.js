// Temporary mock data. Will be replaced with real backend data later.

export const stats = [
  { id: 'clicks', label: 'Clicks total', value: '128,540' },
  { id: 'bots', label: 'Bots blocked', value: '37,219', danger: true },
  { id: 'ratio', label: 'Bot ratio', value: '29%' },
  { id: 'rps', label: 'Requests / sec', value: '1,842' },
]

export const recentClicks = [
  { ip: '192.168.4.21', agent: 'Chrome/124 Win', status: 'human' },
  { ip: '10.0.55.8', agent: 'python-requests/2.31', status: 'bot' },
  { ip: '172.16.9.103', agent: 'Safari/17 iPhone', status: 'human' },
  { ip: '10.0.55.9', agent: 'curl/8.1', status: 'bot' },
  { ip: '192.168.1.44', agent: 'Firefox/126 Linux', status: 'human' },
]

export const blacklist = [
  { ip: '10.0.55.8', reason: 'High frequency', blockedAt: '2026-06-02 14:31' },
  { ip: '10.0.55.9', reason: 'Known bot UA', blockedAt: '2026-06-02 14:28' },
  { ip: '203.0.113.7', reason: 'Rate limit exceeded', blockedAt: '2026-06-02 13:55' },
]