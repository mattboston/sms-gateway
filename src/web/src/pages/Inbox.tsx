import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '@/lib/api';
import Pagination from '@/components/Pagination';
import { usePaginatedList, type Message } from '@/lib/usePaginatedList';
import { textDirection } from '@/lib/text';

function formatRelativeTime(dateStr: string): string {
  const now = new Date();
  const date = new Date(dateStr);
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);

  if (diffSec < 60) return 'just now';
  if (diffMin < 60) return `${diffMin} minute${diffMin !== 1 ? 's' : ''} ago`;
  if (diffHour < 24) return `${diffHour} hour${diffHour !== 1 ? 's' : ''} ago`;
  if (diffDay < 7) return `${diffDay} day${diffDay !== 1 ? 's' : ''} ago`;
  return date.toLocaleDateString();
}

function statusBadge(status: string) {
  switch (status) {
    case 'received':
      return 'bg-blue-100 text-blue-800 dark:bg-[#1f3e52] dark:text-[#268bd2]';
    case 'read':
      return 'bg-gray-100 text-gray-600 dark:bg-[#586e75] dark:text-[#93a1a1]';
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-[#586e75] dark:text-[#eee8d5]';
  }
}

function statusLabel(status: string) {
  if (status === 'received') return 'Unread';
  if (status === 'read') return 'Read';
  return status;
}

export default function Inbox() {
  const navigate = useNavigate();
  const {
    items: messages,
    total,
    page,
    pageSize,
    totalPages,
    loading,
    refreshing,
    error: loadError,
    setPage,
    setPageSize,
    removeItems,
    mapItems,
  } = usePaginatedList<Message>('/sms/inbox', { all: 'true' });

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [deleting, setDeleting] = useState(false);
  const [markingRead, setMarkingRead] = useState(false);
  const [actionError, setActionError] = useState('');

  const error = actionError || loadError;

  // Selection is scoped to the visible page, so leaving the page must clear it —
  // otherwise Delete would act on rows the user can no longer see.
  useEffect(() => {
    setSelected(new Set());
  }, [page, pageSize]);

  const toggleSelect = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleAll = () => {
    if (selected.size === messages.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(messages.map((m) => m.id)));
    }
  };


  const selectedUnreadIds = messages
    .filter((m) => selected.has(m.id) && m.status === 'received')
    .map((m) => m.id);

  const handleMarkRead = async () => {
    if (selectedUnreadIds.length === 0) return;

    setMarkingRead(true);
    setActionError('');
    const marked = new Set<string>();
    try {
      const results = await Promise.allSettled(
        selectedUnreadIds.map((id) => api.put(`/sms/${id}/read`).then(() => id)),
      );
      for (const result of results) {
        if (result.status === 'fulfilled') marked.add(result.value);
      }
      if (marked.size !== selectedUnreadIds.length) {
        setActionError('Failed to mark some messages as read.');
      }
    } finally {
      mapItems((prev) =>
        prev.map((m) => (marked.has(m.id) ? { ...m, status: 'read' } : m)),
      );
      setSelected((prev) => {
        const next = new Set(prev);
        marked.forEach((id) => next.delete(id));
        return next;
      });
      setMarkingRead(false);
    }
  };

  const handleDelete = async () => {
    if (selected.size === 0) return;
    if (!confirm(`Delete ${selected.size} message${selected.size !== 1 ? 's' : ''}?`)) return;

    setDeleting(true);
    setActionError('');
    const deleted = new Set<string>();
    try {
      // Deletes run concurrently; sequential awaits made bulk deletes take one
      // round trip per message.
      const results = await Promise.allSettled(
        [...selected].map((id) => api.delete(`/sms/${id}`).then(() => id)),
      );
      for (const result of results) {
        if (result.status === 'fulfilled') deleted.add(result.value);
      }
      if (deleted.size !== selected.size) {
        setActionError('Failed to delete some messages.');
      }
    } finally {
      removeItems(deleted);
      setSelected((prev) => {
        const next = new Set(prev);
        deleted.forEach((id) => next.delete(id));
        return next;
      });
      setDeleting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <p className="text-gray-500 dark:text-[#93a1a1]">Loading inbox...</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-baseline gap-3">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-[#fdf6e3]">Inbox</h1>
          {total > 0 && (
            <span className="text-sm text-gray-500 dark:text-[#93a1a1]">
              {total.toLocaleString()} message{total !== 1 ? 's' : ''}
            </span>
          )}
        </div>
        {selected.size > 0 && (
          <div className="flex items-center gap-2">
            {selectedUnreadIds.length > 0 && (
              <button
                onClick={handleMarkRead}
                disabled={markingRead || deleting}
                className="inline-flex items-center rounded-md bg-gray-100 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-200 disabled:opacity-50 dark:bg-[#073642] dark:text-[#93a1a1] dark:hover:bg-[#0a4452] dark:ring-1 dark:ring-[#586e75]"
              >
                {markingRead
                  ? 'Marking...'
                  : `Mark as Read (${selectedUnreadIds.length})`}
              </button>
            )}
            <button
              onClick={handleDelete}
              disabled={deleting || markingRead}
              className="inline-flex items-center rounded-md bg-red-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
            >
              {deleting ? 'Deleting...' : `Delete (${selected.size})`}
            </button>
          </div>
        )}
      </div>

      {error && (
        <div className="rounded-md bg-red-50 p-3 text-sm text-red-700 dark:bg-[#3b1f23] dark:text-[#dc322f]">
          {error}
        </div>
      )}

      <div className="rounded-lg bg-white shadow-sm dark:bg-[#073642] dark:ring-1 dark:ring-[#586e75]">
        {messages.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <svg
              className="h-12 w-12 text-gray-300 dark:text-[#586e75]"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
              />
            </svg>
            <p className="mt-3 text-sm font-medium text-gray-900 dark:text-[#fdf6e3]">
              No messages
            </p>
            <p className="mt-1 text-sm text-gray-500 dark:text-[#93a1a1]">Your inbox is empty.</p>
          </div>
        ) : (
          <div className={refreshing ? 'opacity-60 transition-opacity' : 'transition-opacity'}>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-200 text-left text-xs font-medium uppercase text-gray-500 dark:border-[#586e75] dark:text-[#93a1a1]">
                    <th className="px-3 py-3 w-10">
                      <input
                        type="checkbox"
                        checked={selected.size === messages.length && messages.length > 0}
                        onChange={toggleAll}
                        aria-label="Select all messages on this page"
                        title="Select all messages on this page"
                        className="rounded border-gray-300"
                      />
                    </th>
                    <th className="px-5 py-3">From</th>
                    <th className="px-5 py-3">Message</th>
                    <th className="px-5 py-3">Status</th>
                    <th className="px-5 py-3">Time</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100 dark:divide-[#586e75]">
                  {messages.map((msg) => (
                    <tr
                      key={msg.id}
                      className={`cursor-pointer transition-colors hover:bg-gray-50 dark:hover:bg-[#0a4452] ${msg.status === 'received' ? 'bg-blue-50/40 dark:bg-[#1f3e52]/40' : ''}`}
                    >
                      <td className="px-3 py-4" onClick={(e) => e.stopPropagation()}>
                        <input
                          type="checkbox"
                          checked={selected.has(msg.id)}
                          onChange={() => toggleSelect(msg.id)}
                          className="rounded border-gray-300"
                        />
                      </td>
                      <td
                        className={`px-5 py-4 whitespace-nowrap ${msg.status === 'received' ? 'font-bold text-gray-900 dark:text-[#fdf6e3]' : 'font-medium text-gray-500 dark:text-[#93a1a1]'}`}
                        onClick={() => navigate(`/messages/${msg.id}`)}
                      >
                        {msg.phone_number}
                      </td>
                      <td
                      dir={textDirection(msg.body)}
                        className={`px-5 py-4 max-w-md truncate ${msg.status === 'received' ? 'font-semibold text-gray-900 dark:text-[#fdf6e3]' : 'text-gray-500 dark:text-[#93a1a1]'}`}
                        onClick={() => navigate(`/messages/${msg.id}`)}
                      >
                        {msg.body}
                      </td>
                      <td className="px-5 py-4" onClick={() => navigate(`/messages/${msg.id}`)}>
                        <span
                          className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${statusBadge(msg.status)}`}
                        >
                          {statusLabel(msg.status)}
                        </span>
                      </td>
                      <td
                        className="px-5 py-4 text-gray-400 whitespace-nowrap dark:text-[#93a1a1]"
                        onClick={() => navigate(`/messages/${msg.id}`)}
                      >
                        {formatRelativeTime(msg.created_at)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <Pagination
              page={page}
              pageSize={pageSize}
              total={total}
              totalPages={totalPages}
              busy={refreshing}
              itemLabel="Messages"
              onPageChange={setPage}
              onPageSizeChange={setPageSize}
            />
          </div>
        )}
      </div>
    </div>
  );
}
