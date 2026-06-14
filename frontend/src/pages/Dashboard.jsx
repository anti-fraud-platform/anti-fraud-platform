import Layout from '../components/Layout';
import { useStats } from '../hooks/useStats';
import StatCard from '../components/StatCard';
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

  const statItems = data
    ? [
        { label: 'Total clicks', value: formatNumber(data.total_clicks), danger: false },
        { label: 'Blocked bots', value: formatNumber(data.blocked_bots), danger: true },
        { label: 'Money saved', value: formatMoney(data.saved_money_usd), danger: false },
      ]
    : [];

  return (
    <Layout title="Dashboard">
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
            {statItems.map((item, idx) => (
              <StatCard key={idx} label={item.label} value={item.value} danger={item.danger} />
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