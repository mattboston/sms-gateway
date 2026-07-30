import { PAGE_SIZE_OPTIONS } from '@/lib/usePaginatedList';

interface PaginationProps {
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
  busy?: boolean;
  /** Plural noun for the paged items, used in the screen-reader label. */
  itemLabel?: string;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
}

const buttonClass =
  'rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:cursor-not-allowed disabled:opacity-40 dark:border-[#586e75] dark:bg-[#073642] dark:text-[#93a1a1] dark:hover:bg-[#0a4452] dark:focus:ring-[#268bd2]';

/**
 * Page navigation for a message list.
 *
 * Renders nothing when everything already fits on one page, so small mailboxes
 * are not cluttered with controls that cannot do anything.
 */
export default function Pagination({
  page,
  pageSize,
  total,
  totalPages,
  busy = false,
  itemLabel = 'items',
  onPageChange,
  onPageSizeChange,
}: PaginationProps) {
  if (total <= PAGE_SIZE_OPTIONS[0] && totalPages <= 1) return null;

  const first = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const last = Math.min(page * pageSize, total);

  return (
    <div className="flex flex-col gap-3 border-t border-gray-200 px-5 py-3 sm:flex-row sm:items-center sm:justify-between dark:border-[#586e75]">
      <div className="flex items-center gap-3 text-sm text-gray-500 dark:text-[#93a1a1]">
        <span>
          {first.toLocaleString()}&ndash;{last.toLocaleString()} of {total.toLocaleString()}
        </span>
        {/*
          `relative` is load-bearing: sr-only is position:absolute, and without a
          positioned ancestor its containing block is the initial containing
          block rather than this bar. It then escapes the scrolling <main>,
          lands at its static position deep in the document, and stretches the
          page to that height — leaving a tall blank area below the layout.
        */}
        <label className="relative flex items-center gap-2">
          <span className="sr-only">{itemLabel} per page</span>
          <select
            value={pageSize}
            onChange={(e) => onPageSizeChange(Number(e.target.value))}
            className="rounded-md border border-gray-300 bg-white px-2 py-1 text-sm text-gray-700 outline-none focus:ring-2 focus:ring-blue-500 dark:border-[#586e75] dark:bg-[#073642] dark:text-[#93a1a1] dark:focus:ring-[#268bd2]"
          >
            {PAGE_SIZE_OPTIONS.map((size) => (
              <option key={size} value={size}>
                {size} per page
              </option>
            ))}
          </select>
        </label>
      </div>

      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1 || busy}
          className={buttonClass}
        >
          Previous
        </button>
        <span className="text-sm text-gray-500 dark:text-[#93a1a1]" aria-live="polite">
          Page {page.toLocaleString()} of {totalPages.toLocaleString()}
        </span>
        <button
          type="button"
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages || busy}
          className={buttonClass}
        >
          Next
        </button>
      </div>
    </div>
  );
}
