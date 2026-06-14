import { useState, useEffect } from 'react';
import Layout from '../components/Layout';
import { blacklist } from '../data/mockData';
import SkeletonBlacklistRow from '../components/SkeletonBlacklistRow';

function Blacklist() {
  const [loading, setLoading] = useState(true);

  // For testing purposes
  useEffect(() => {
    const timer = setTimeout(() => setLoading(false), 2000);
    return () => clearTimeout(timer);
  }, []);

  return (
    <Layout title="Blacklist">
      <div className="border border-border rounded-xl overflow-hidden">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="bg-surface text-left text-text-muted">
              <th className="px-3.5 py-2.5 font-medium">IP</th>
              <th className="px-3.5 py-2.5 font-medium">Reason</th>
              <th className="px-3.5 py-2.5 font-medium">Blocked at</th>
            </tr>
          </thead>
          <tbody>
            {loading
              ? Array.from({ length: 3 }).map((_, i) => <SkeletonBlacklistRow key={i} />)
              : blacklist.map((row, i) => (
                  <tr key={i} className="border-t border-border">
                    <td className="px-3.5 py-2.5 font-mono">{row.ip}</td>
                    <td className="px-3.5 py-2.5 text-text-muted">{row.reason}</td>
                    <td className="px-3.5 py-2.5 text-text-muted">{row.blockedAt}</td>
                  </tr>
                ))}
          </tbody>
        </table>
      </div>
    </Layout>
  );
}

export default Blacklist;