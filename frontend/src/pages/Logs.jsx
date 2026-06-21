import { useState, useMemo } from 'react';
import Layout from '../components/Layout';
import { useLogs } from '../hooks/useLogs';
import SkeletonLogsRow from '../components/SkeletonLogsRow';
import LogFilters from '../components/LogFilters';
import TablePagination from '../components/TablePagination';

function formatDateTime(isoString) {
  const d = new Date(isoString);
  return d.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

function formatReason(rawReason) {
  switch (rawReason) {
    case 'allowed':
      return 'Allowed';
    case 'static_blacklist':
      return 'Blacklist';
    case 'rate_limit_exceeded':
      return 'Rate limit exceeded';
    default:
      return rawReason || '—';
  }
}

const logsTableHeader = (
  <thead>
    <tr className="bg-surface text-left text-text-muted">
      <th className="px-3.5 py-2.5 font-medium">IP</th>
      <th className="px-3.5 py-2.5 font-medium">Campaign</th>
      <th className="px-3.5 py-2.5 font-medium">User-Agent</th>
      <th className="px-3.5 py-2.5 font-medium">Reason</th>
      <th className="px-3.5 py-2.5 font-medium">Date/Time</th>
    </tr>
  </thead>
);

function Logs() {
  // Pagination state.
  const [page, setPage] = useState(1);
  const limit = 20;

  // Filter state.
  const [campaignId, setCampaignId] = useState('');
  const [isBotFilter, setIsBotFilter] = useState(''); // '', 'true', 'false'
  const [reasonFilter, setReasonFilter] = useState('');

  // Preparing params.
  const params = useMemo(() => {
    const p = { page, limit };
    if (campaignId.trim() !== '') p.campaign_id = campaignId.trim();
    if (isBotFilter !== '') p.is_bot = isBotFilter === 'true';
    if (reasonFilter.trim() !== '') p.reason = reasonFilter.trim();
    return p;
  }, [page, limit, campaignId, isBotFilter, reasonFilter]);

  const { data, loading, error } = useLogs(params);

  const logEntries = data?.data ?? [];
  const totalPages = data?.total_pages ?? 1;

  const handleCampaignChange = (newValue) => {
    setCampaignId(newValue);
    setPage(1);
  };

  const handleReasonChange = (newReason) => {
    setReasonFilter(newReason);
    setPage(1);
    if (newReason === 'allowed') {
      setIsBotFilter('false');
    } else if (newReason !== '') {
      setIsBotFilter('true');
    }
  };

  const handleTypeChange = (newType) => {
    setIsBotFilter(newType);
    setPage(1);
    if (newType === 'true' && reasonFilter === 'allowed') {
      setReasonFilter('');
    } else if (newType === 'false' && reasonFilter !== '' && reasonFilter !== 'allowed') {
      setReasonFilter('');
    }
  };

  const handlePageChange = (newPage) => {
    setPage(newPage);
  };

  return (
    <Layout title="Logs">
      <LogFilters
        campaignId={campaignId}
        isBotFilter={isBotFilter}
        reasonFilter={reasonFilter}
        onCampaignChange={handleCampaignChange}
        onTypeChange={handleTypeChange}
        onReasonChange={handleReasonChange}
      />

      {loading && !data && (
        <div className="border border-border rounded-xl overflow-hidden">
          <table className="w-full border-collapse text-sm">
            {logsTableHeader}
            <tbody>
              {Array.from({ length: 5 }).map((_, i) => (
                <SkeletonLogsRow key={i} />
              ))}
            </tbody>
          </table>
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
            <p className="text-[#9a6b00] text-xs mb-4">
              Connection issue - showing last known values.
            </p>
          )}

          <div className="border border-border rounded-xl overflow-hidden">
            <table className="w-full border-collapse text-sm">
              {logsTableHeader}
              <tbody>
                {logEntries.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-3.5 py-6 text-center text-text-muted">
                      No logs yet.
                    </td>
                  </tr>
                ) : (
                  logEntries.map((entry) => (
                    <tr
                      key={entry.id}
                      className={`border-t border-border ${
                        entry.is_bot
                          ? entry.reason === 'static_blacklist'
                            ? 'bg-red-200'
                            : entry.reason === 'rate_limit_exceeded'
                            ? 'bg-orange-100'
                            : 'bg-red-100'
                          : ''
                      }`}
                    >
                      <td className="px-3.5 py-2.5 font-mono">{entry.ip}</td>
                      <td className="px-3.5 py-2.5 font-mono">{entry.campaign_id}</td>
                      <td className="px-3.5 py-2.5 text-text-muted truncate max-w-[200px]">
                        {entry.user_agent}
                      </td>
                      <td className="px-3.5 py-2.5 text-text-muted text-xs">
                        {formatReason(entry.reason)}
                      </td>
                      <td className="px-3.5 py-2.5 text-text-muted text-xs whitespace-nowrap">
                        {formatDateTime(entry.processed_at)}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          <TablePagination
            page={data?.page ?? page}
            totalPages={totalPages}
            onPageChange={handlePageChange}
          />
        </>
      )}
    </Layout>
  );
}

export default Logs;