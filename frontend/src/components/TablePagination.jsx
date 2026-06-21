import { useRef } from 'react';

function TablePagination({ page, totalPages, limit, onPageChange, onLimitChange }) {
  const jumpRef = useRef(null);

  const handleJump = () => {
    const raw = jumpRef.current?.value;
    if (!raw) return;
    const parsed = parseInt(raw, 10);
    if (!isNaN(parsed)) {
      const clamped = Math.max(1, Math.min(parsed, totalPages));
      onPageChange(clamped);
      jumpRef.current.value = '';
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter') handleJump();
  };

  const handleLimitChange = (e) => {
    const newLimit = parseInt(e.target.value, 10);
    onLimitChange(newLimit);
  };

  return (
    <div className="flex items-center justify-between mt-4 text-sm px-2">
      <div className="flex items-center gap-4">
        <button
          onClick={() => onPageChange(Math.max(1, page - 1))}
          disabled={page <= 1}
          className="px-3 py-1.5 rounded-md border border-border disabled:opacity-50"
        >
          Previous
        </button>

        <div className="flex items-center gap-2 text-text-muted">
          <span>Page {page} of {totalPages}</span>
          <input
            ref={jumpRef}
            type="text"
            placeholder="Jump to…"
            onKeyDown={handleKeyDown}
            className="w-20 px-2 py-1 text-sm border border-border rounded-md bg-white"
          />
          <button
            onClick={handleJump}
            className="px-2 py-1 rounded-md border border-border hover:bg-gray-50"
          >
            Go
          </button>
        </div>

        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
          className="px-3 py-1.5 rounded-md border border-border disabled:opacity-50"
        >
          Next
        </button>
      </div>

      <div className="flex items-center gap-2 text-text-muted ml-4">
        <label htmlFor="limit-select" className="text-xs">
          Per page
        </label>
        <select
          id="limit-select"
          value={limit}
          onChange={handleLimitChange}
          className="px-2 py-1 text-sm border border-border rounded-md bg-white"
        >
          <option value="10">10</option>
          <option value="20">20</option>
          <option value="50">50</option>
          <option value="100">100</option>
        </select>
      </div>
    </div>
  );
}

export default TablePagination;