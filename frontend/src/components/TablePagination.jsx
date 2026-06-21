import { useRef } from 'react';

function TablePagination({ page, totalPages, onPageChange }) {
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

  return (
    <div className="flex items-center justify-between mt-4 text-sm">
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
  );
}

export default TablePagination;