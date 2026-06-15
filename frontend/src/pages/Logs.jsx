import { useState } from 'react';
import Layout from '../components/Layout';
import { recentClicks } from '../data/mockData';
import SkeletonLogsRow from '../components/SkeletonLogsRow';

function Logs() {
  const [loading/*, setLoading*/] = useState(false);

  return (
    <Layout title="Logs">
      <div className="border border-border rounded-xl overflow-hidden">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="bg-surface text-left text-text-muted">
              <th className="px-3.5 py-2.5 font-medium">IP</th>
              <th className="px-3.5 py-2.5 font-medium">User-Agent</th>
              <th className="px-3.5 py-2.5 font-medium text-center">Status</th>
            </tr>
          </thead>
          <tbody>
            {loading
              ? Array.from({ length: 5 }).map((_, i) => <SkeletonLogsRow key={i} />)
              : recentClicks.map((row, i) => (
                  <tr key={i} className="border-t border-border">
                    <td className="px-3.5 py-2.5 font-mono">{row.ip}</td>
                    <td className="px-3.5 py-2.5 text-text-muted">{row.agent}</td>
                    <td className="px-3.5 py-2.5 text-center">
                      <span
                        className={`inline-block px-2.5 py-0.5 rounded-lg text-xs ${
                          row.status === 'bot'
                            ? 'bg-danger-light text-danger'
                            : 'bg-success-light text-success'
                        }`}
                      >
                        {row.status}
                      </span>
                    </td>
                  </tr>
                ))}
          </tbody>
        </table>
      </div>
    </Layout>
  );
}

export default Logs;