function LogFilters({
  campaignId,
  isBotFilter,
  reasonFilter,
  onCampaignChange,
  onTypeChange,
  onReasonChange,
}) {
  return (
    <div className="flex flex-wrap items-end gap-3 mb-4">
      <div className="flex flex-col gap-1">
        <label className="text-xs text-text-muted">Campaign</label>
        <input
          type="text"
          placeholder="All campaigns"
          value={campaignId}
          onChange={(e) => onCampaignChange(e.target.value)}
          className="px-3 py-1.5 text-sm border border-border rounded-md bg-white"
        />
      </div>

      <div className="flex flex-col gap-1">
        <label className="text-xs text-text-muted">Type</label>
        <select
          value={isBotFilter}
          onChange={(e) => onTypeChange(e.target.value)}
          className="px-3 py-1.5 text-sm border border-border rounded-md bg-white"
        >
          <option value="">All traffic</option>
          <option value="true">Bots only</option>
          <option value="false">Humans only</option>
        </select>
      </div>

      <div className="flex flex-col gap-1">
        <label className="text-xs text-text-muted">Reason</label>
        <select
          value={reasonFilter}
          onChange={(e) => onReasonChange(e.target.value)}
          className="px-3 py-1.5 text-sm border border-border rounded-md bg-white"
        >
          <option value="">All reasons</option>
          <option value="allowed">Allowed</option>
          <option value="static_blacklist">Blacklist</option>
          <option value="rate_limit_exceeded">Rate limit exceeded</option>
        </select>
      </div>
    </div>
  );
}

export default LogFilters;