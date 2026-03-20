import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '@/lib/api';

interface Message {
  id: string;
  direction: string;
  phone_number: string;
  body: string;
  status: string;
  created_at: string;
}

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
    case 'pending':
      return 'bg-yellow-100 text-yellow-800 dark:bg-[#3b3200] dark:text-[#b58900]';
    case 'sending':
      return 'bg-blue-100 text-blue-800 dark:bg-[#1f3e52] dark:text-[#268bd2]';
    case 'sent':
      return 'bg-green-100 text-green-800 dark:bg-[#213a25] dark:text-[#859900]';
    case 'failed':
      return 'bg-red-100 text-red-800 dark:bg-[#3b1f23] dark:text-[#dc322f]';
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-[#586e75] dark:text-[#eee8d5]';
  }
}

export default function Outbox() {
  const navigate = useNavigate();
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    const fetchOutbox = async () => {
      try {
        const res = await api.get<Message[]>('/sms/outbox');
        setMessages(res.data);
      } catch {
        setError('Failed to load outbox messages.');
      } finally {
        setLoading(false);
      }
    };
    fetchOutbox();
  }, []);

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

  const handleDelete = async () => {
    if (selected.size === 0) return;
    if (!confirm(`Delete ${selected.size} message${selected.size !== 1 ? 's' : ''}?`)) return;

    setDeleting(true);
    setError('');
    const deleted = new Set<string>();
    try {
      for (const id of selected) {
        await api.delete(`/sms/${id}`);
        deleted.add(id);
      }
    } catch {
      setError('Failed to delete some messages.');
    } finally {
      setMessages((prev) => prev.filter((m) => !deleted.has(m.id)));
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
        <p className="text-gray-500 dark:text-[#93a1a1]">Loading outbox...</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-[#fdf6e3]">Outbox</h1>
        {selected.size > 0 && (
          <button
            onClick={handleDelete}
            disabled={deleting}
            className="inline-flex items-center rounded-md bg-red-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
          >
            {deleting ? 'Deleting...' : `Delete (${selected.size})`}
          </button>
        )}
      </div>

      {error && (
        <div className="rounded-md bg-red-50 p-3 text-sm text-red-700 dark:bg-[#3b1f23] dark:text-[#dc322f]">{error}</div>
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
                d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
              />
            </svg>
            <p className="mt-3 text-sm font-medium text-gray-900 dark:text-[#fdf6e3]">No messages</p>
            <p className="mt-1 text-sm text-gray-500 dark:text-[#93a1a1]">Your outbox is empty.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 text-left text-xs font-medium uppercase text-gray-500 dark:border-[#586e75] dark:text-[#93a1a1]">
                  <th className="px-3 py-3 w-10">
                    <input
                      type="checkbox"
                      checked={selected.size === messages.length && messages.length > 0}
                      onChange={toggleAll}
                      className="rounded border-gray-300"
                    />
                  </th>
                  <th className="px-5 py-3">To</th>
                  <th className="px-5 py-3">Message</th>
                  <th className="px-5 py-3">Status</th>
                  <th className="px-5 py-3">Time</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-[#586e75]">
                {messages.map((msg) => (
                  <tr
                    key={msg.id}
                    className="cursor-pointer transition-colors hover:bg-gray-50 dark:hover:bg-[#0a4452]"
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
                      className="px-5 py-4 font-medium text-gray-900 whitespace-nowrap dark:text-[#eee8d5]"
                      onClick={() => navigate(`/messages/${msg.id}`)}
                    >
                      {msg.phone_number}
                    </td>
                    <td
                      className="px-5 py-4 text-gray-600 max-w-md truncate dark:text-[#93a1a1]"
                      onClick={() => navigate(`/messages/${msg.id}`)}
                    >
                      {msg.body}
                    </td>
                    <td
                      className="px-5 py-4"
                      onClick={() => navigate(`/messages/${msg.id}`)}
                    >
                      <span
                        className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${statusBadge(msg.status)}`}
                      >
                        {msg.status}
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
        )}
      </div>
    </div>
  );
}
