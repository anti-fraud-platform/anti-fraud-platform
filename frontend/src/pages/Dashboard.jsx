import { useState, useEffect } from 'react';
import Layout from '../components/Layout';
import { stats } from '../data/mockData';
import SkeletonCard from '../components/SkeletonCard';
import SkeletonChart from '../components/SkeletonChart';

function Dashboard() {
  const [loading, setLoading] = useState(true);

  // For testing purposes
  useEffect(() => {
    const timer = setTimeout(() => setLoading(false), 2000);
    return () => clearTimeout(timer);
  }, []);

  const dashboardStats = [
    stats[0],                                   // Clicks total
    stats[1],                                   // Bots blocked
    { id: 'saved', label: 'Saved Money', value: '$45,600', danger: false },
  ];

  return (
    <Layout title="Dashboard">
      {/* Stat cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 mb-4">
        {loading
          ? Array.from({ length: 3 }).map((_, i) => <SkeletonCard key={i} />)
          : dashboardStats.map((s) => (
              <div key={s.id} className="bg-surface rounded-lg p-4 text-center">
                <p className="text-xs text-text-muted mb-1.5">{s.label}</p>
                <p
                  className={`text-2xl font-semibold ${
                    s.danger ? 'text-danger' : 'text-text-main'
                  }`}
                >
                  {s.value}
                </p>
              </div>
            ))}
      </div>

      {/* Chart placeholders */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        {loading ? (
          <>
            <SkeletonChart />
            <SkeletonChart />
          </>
        ) : (
          <>
            <div className="bg-chart-bg border border-chart-bar rounded-xl h-52 flex items-end justify-center gap-2 px-4 py-6">
              <div className="w-[8%] bg-chart-bar rounded h-16" />
              <div className="w-[8%] bg-chart-bar rounded h-24" />
              <div className="w-[8%] bg-chart-bar rounded h-20" />
              <div className="w-[8%] bg-chart-bar rounded h-10" />
              <div className="w-[8%] bg-chart-bar rounded h-28" />
              <div className="w-[8%] bg-chart-bar rounded h-14" />
              <div className="w-[8%] bg-chart-bar rounded h-18" />
              <div className="w-[8%] bg-chart-bar rounded h-22" />
            </div>
            <div className="bg-chart-bg border border-chart-bar rounded-xl h-52 flex items-end justify-center gap-2 px-4 py-6">
              <div className="w-[8%] bg-chart-bar rounded h-16" />
              <div className="w-[8%] bg-chart-bar rounded h-24" />
              <div className="w-[8%] bg-chart-bar rounded h-20" />
              <div className="w-[8%] bg-chart-bar rounded h-10" />
              <div className="w-[8%] bg-chart-bar rounded h-28" />
              <div className="w-[8%] bg-chart-bar rounded h-14" />
              <div className="w-[8%] bg-chart-bar rounded h-18" />
              <div className="w-[8%] bg-chart-bar rounded h-22" />
            </div>
          </>
        )}
      </div>
    </Layout>
  );
}

export default Dashboard;