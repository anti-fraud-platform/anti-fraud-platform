import { MousePointerClick, ShieldAlert, CheckCircle, Flag } from 'lucide-react';
import Layout from '../components/Layout';
import { useStats } from '../hooks/useStats';
import StatCard from '../components/StatCard';
import SkeletonCard from '../components/SkeletonCard';
import ReasonBreakdownChart from '../components/ReasonBreakdownChart';
import PipelineEffectiveness from '../components/PipelineEffectiveness';
import DetectionPipeline from '../components/DetectionPipeline';
import TopCampaigns from '../components/TopCampaigns';
import CampaignCostBreakdown from '../components/CampaignCostBreakdown';
import TrafficOverTime from '../components/TrafficOverTime';
import BlockedByReason from '../components/BlockedByReason';
import RecentDetections from '../components/RecentDetections';
import { useTrend } from '../hooks/useTrend';
import TopAttackingIPs from '../components/TopAttackingIPs';
import RecentActivity from '../components/RecentActivity';
import SystemHealth from '../components/SystemHealth';

function formatNumber(n) {
  return Number(n).toLocaleString('en-US');
}

function Dashboard() {
  const { data, loading, error } = useStats(2500);
  const trend = useTrend(5000);

  const statItems = data
    ? [
        { label: 'Total clicks', value: formatNumber(data.total_clicks), danger: false, icon: <MousePointerClick/>, delta: '18.4%', deltaUp: true },
        { label: 'Blocked clicks', value: formatNumber(data.blocked_count ?? data.blocked_bots), danger: true, icon: <ShieldAlert/>, delta: '24.6%', deltaUp: true },
        { label: 'Allowed clicks', value: formatNumber(data.allowed_count ?? 0), danger: false, icon: <CheckCircle/>, delta: '11.2%', deltaUp: true },
        { label: 'Active campaigns', value: formatNumber((data.campaigns ?? []).length), danger: false, icon: <Flag/>, delta: '2', deltaUp: true },
      ]
    : [];

  const campaigns = data?.campaigns ?? [];
  const topBlockedIPs = data?.top_blocked_ips ?? [];
  const reasonBreakdown = data?.reason_breakdown ?? {};

  return (
    <Layout title="Dashboard" error={error}>
      {loading && !data && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
        </div>
      )}

      {error && !data && (
        <p className="text-danger">
          Analytics backend is unavailable. Retrying every few seconds...
        </p>
      )}

      {data && (
        <>
          {error && (
            <p className="text-[#9a6b00] text-xs mb-3">
              Connection issue - showing last known values.
            </p>
          )}

          <div className="grid grid-cols-1 xl:grid-cols-4 gap-4">
            {/* LEFT: wide main column */}
            <div className="xl:col-span-3 flex flex-col gap-4">
              {/* Stat cards */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
                {statItems.map((item, idx) => (
                  <StatCard
                    key={idx}
                    label={item.label}
                    value={item.value}
                    danger={item.danger}
                    icon={item.icon}
                    delta={item.delta}
                    deltaUp={item.deltaUp}
                  />
                ))}
              </div>

              {/* Detection pipeline */}
              <DetectionPipeline data={data} />

              {/* Charts + top campaigns row */}
              <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
                <TrafficOverTime trend={trend} />
                <BlockedByReason />
                <CampaignCostBreakdown campaigns={campaigns} />
              </div>

              {/* Top campaigns */}
              <TopCampaigns campaigns={campaigns} />

              {/* Recent detections */}
              <RecentDetections />
            </div>

            {/* RIGHT: narrow rail */}
            <div className="flex flex-col gap-4 h-full">
              {/* Reason breakdown donut */}
              <ReasonBreakdownChart reasonBreakdown={reasonBreakdown} />

              {/* Pipeline effectiveness */}
              <PipelineEffectiveness reasonBreakdown={reasonBreakdown} totalClicks={data.total_clicks} />

              {/* Top attacking IPs */}
              <TopAttackingIPs topBlockedIPs={topBlockedIPs} />

              {/* Recent activity */}
              <RecentActivity />

              {/* System health — stretches to fill remaining height */}
              <div className="flex-1 flex flex-col">
                <SystemHealth />
              </div>
            </div>
          </div>
        </>
      )}
    </Layout>
  );
}

export default Dashboard;