import Layout from '../components/Layout';
import { useStats } from '../hooks/useStats';
import SkeletonCard from '../components/SkeletonCard';
import SkeletonChart from '../components/SkeletonChart';

function formatNumber(n) {
  return Number(n).toLocaleString('en-US');
}

function formatMoney(n) {
  return '$' + Number(n).toLocaleString('en-US', { maximumFractionDigits: 0 });
}

function Dashboard() {
  const { data, loading, error } = useStats(2500);

  const dashboardStats = data
    ? [
        { id: 'clicks', label: 'Total clicks', value: formatNumber(data.total_clicks), danger: false },
        { id: 'bots', label: 'Blocked bots', value: formatNumber(data.blocked_bots), danger: true },
        { id: 'saved', label: 'Money saved', value: formatMoney(data.saved_money_usd), danger: false },
      ]
    : [];

  return (
    <Layout title="Dashboard">
      {}
      {loading && !data && (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 mb-4">
            <SkeletonCard />
            <SkeletonCard />
            <SkeletonCard />
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <SkeletonChart />
            <SkeletonChart />
          </div>
        </>
      )}

      {error && !data && (
        <p className="text-danger">
          Analytics backend is unavailable. Retrying every few seconds...
        </p>
      )}

      {data && (
        <>
          {error && (
            <p className="text-[#9a6b00] text-xs mt-0">
              Connection issue — showing last known values.
            </p>
          )}

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 mb-4">
            {dashboardStats.map((s) => (
              <div key={s.id} className="bg-surface rounded-lg p-4 text-center">
                <p className="text-xs text-text-muted mb-1.5">{s.label}</p>
                <p className={`text-2xl font-semibold ${s.danger ? 'text-danger' : 'text-text-main'}`}>
                  {s.value}
                </p>
              </div>
            ))}
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <SkeletonChart />
            <SkeletonChart />
          </div>
        </>
      )}
    </Layout>
  );
}

export default Dashboard;