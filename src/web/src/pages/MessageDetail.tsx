import { useState, useEffect, type FormEvent } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api from '@/lib/api';

interface Message {
  id: string;
  direction: 'inbound' | 'outbound';
  phone_number: string;
  body: string;
  status: string;
  api_key_id?: string;
  modem_response?: string;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

function formatDateTime(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
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
  const styles: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-800',
    sending: 'bg-blue-100 text-blue-800',
    sent: 'bg-green-100 text-green-800',
    failed: 'bg-red-100 text-red-800',
    received: 'bg-blue-100 text-blue-800',
    read: 'bg-gray-100 text-gray-600',
  };
  return styles[status] ?? 'bg-gray-100 text-gray-800';
}

function statusLabel(status: string) {
  if (status === 'received') return 'Unread';
  if (status === 'read') return 'Read';
  return status;
}

export default function MessageDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [message, setMessage] = useState<Message | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [debugOpen, setDebugOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [replyBody, setReplyBody] = useState('');
  const [replySending, setReplySending] = useState(false);
  const [replyResult, setReplyResult] = useState<{ type: 'success' | 'error'; message: string } | null>(null);
  const [togglingRead, setTogglingRead] = useState(false);

  useEffect(() => {
    const fetchMessage = async () => {
      try {
        const res = await api.get<Message>(`/sms/${id}`);
        setMessage(res.data);
      } catch {
        setError('Message not found.');
      } finally {
        setLoading(false);
      }
    };
    fetchMessage();
  }, [id]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <p className="text-gray-500">Loading message...</p>
      </div>
    );
  }

  if (error || !message) {
    return (
      <div className="space-y-4">
        <button
          onClick={() => navigate(-1)}
          className="text-sm text-blue-600 hover:text-blue-800 transition-colors"
        >
          &larr; Back
        </button>
        <div className="rounded-md bg-red-50 p-4 text-sm text-red-700">
          {error || 'Message not found.'}
        </div>
      </div>
    );
  }

  const backPath = message.direction === 'inbound' ? '/inbox' : '/outbox';
  const backLabel = message.direction === 'inbound' ? 'Inbox' : 'Outbox';
  const phoneLabel = message.direction === 'inbound' ? 'From' : 'To';
  const hasDebugInfo = message.modem_response || message.error_message;

  const handleToggleRead = async () => {
    if (!message) return;
    setTogglingRead(true);
    try {
      const endpoint = message.status === 'received'
        ? `/sms/${message.id}/read`
        : `/sms/${message.id}/unread`;
      await api.put(endpoint);
      setMessage({
        ...message,
        status: message.status === 'received' ? 'read' : 'received',
      });
    } catch {
      setError('Failed to update message status.');
    } finally {
      setTogglingRead(false);
    }
  };

  const handleReply = async (e: FormEvent) => {
    e.preventDefault();
    setReplySending(true);
    setReplyResult(null);
    try {
      const res = await api.post('/sms/send', { to: message.phone_number, body: replyBody });
      if (res.data.status === 'sent') {
        setReplyResult({ type: 'success', message: 'Reply sent successfully.' });
        setReplyBody('');
      } else {
        setReplyResult({ type: 'error', message: res.data.message || 'Failed to send reply.' });
      }
    } catch {
      setReplyResult({ type: 'error', message: 'Failed to send reply.' });
    } finally {
      setReplySending(false);
    }
  };

  const handleDelete = async () => {
    if (!confirm('Delete this message?')) return;
    setDeleting(true);
    try {
      await api.delete(`/sms/${message.id}`);
      navigate(backPath);
    } catch {
      setError('Failed to delete message.');
      setDeleting(false);
    }
  };

  return (
    <div className="space-y-6 max-w-3xl">
      <div className="flex items-center justify-between">
        <button
          onClick={() => navigate(backPath)}
          className="text-sm text-blue-600 hover:text-blue-800 transition-colors"
        >
          &larr; Back to {backLabel}
        </button>
        <div className="flex items-center gap-2">
          {message.direction === 'inbound' && (message.status === 'received' || message.status === 'read') && (
            <button
              onClick={handleToggleRead}
              disabled={togglingRead}
              className="inline-flex items-center rounded-md bg-gray-100 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-200 disabled:opacity-50"
            >
              {togglingRead
                ? 'Updating...'
                : message.status === 'received'
                  ? 'Mark as Read'
                  : 'Mark as Unread'}
            </button>
          )}
          <button
            onClick={handleDelete}
            disabled={deleting}
            className="inline-flex items-center rounded-md bg-red-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
          >
            {deleting ? 'Deleting...' : 'Delete'}
          </button>
        </div>
      </div>

      <div className="rounded-lg bg-white p-6 shadow-sm">
        <div className="flex items-start justify-between">
          <h1 className="text-xl font-bold text-gray-900">Message Detail</h1>
          <span
            className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${statusBadge(message.status)}`}
          >
            {statusLabel(message.status)}
          </span>
        </div>

        <div className="mt-6 space-y-4">
          {/* Direction */}
          <div className="flex items-center gap-3">
            <span className="w-28 shrink-0 text-sm font-medium text-gray-500">Direction</span>
            <span
              className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                message.direction === 'inbound'
                  ? 'bg-blue-100 text-blue-700'
                  : 'bg-gray-100 text-gray-700'
              }`}
            >
              {message.direction === 'inbound' ? 'Inbound' : 'Outbound'}
            </span>
          </div>

          {/* Phone Number */}
          <div className="flex items-center gap-3">
            <span className="w-28 shrink-0 text-sm font-medium text-gray-500">{phoneLabel}</span>
            <span className="text-sm text-gray-900 font-medium">{message.phone_number}</span>
          </div>

          {/* Message Body */}
          <div className="flex gap-3">
            <span className="w-28 shrink-0 text-sm font-medium text-gray-500 pt-0.5">Message</span>
            <p className="text-sm text-gray-900 whitespace-pre-wrap">{message.body}</p>
          </div>

          {/* Created At */}
          <div className="flex items-center gap-3">
            <span className="w-28 shrink-0 text-sm font-medium text-gray-500">Created</span>
            <span className="text-sm text-gray-900">
              {formatDateTime(message.created_at)}{' '}
              <span className="text-gray-400">({formatRelativeTime(message.created_at)})</span>
            </span>
          </div>

          {/* Updated At */}
          <div className="flex items-center gap-3">
            <span className="w-28 shrink-0 text-sm font-medium text-gray-500">Updated</span>
            <span className="text-sm text-gray-900">
              {formatDateTime(message.updated_at)}{' '}
              <span className="text-gray-400">({formatRelativeTime(message.updated_at)})</span>
            </span>
          </div>

          {/* Message ID */}
          <div className="flex items-center gap-3">
            <span className="w-28 shrink-0 text-sm font-medium text-gray-500">ID</span>
            <span className="text-sm text-gray-500 font-mono">{message.id}</span>
          </div>
        </div>
      </div>

      {/* Reply Section */}
      {message.direction === 'inbound' && (
        <div className="rounded-lg bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-gray-800">Reply</h2>
          <p className="mt-1 text-sm text-gray-500">
            Replying to {message.phone_number}
          </p>
          {replyResult && (
            <div
              className={`mt-3 rounded-md p-3 text-sm ${
                replyResult.type === 'success'
                  ? 'bg-green-50 text-green-700'
                  : 'bg-red-50 text-red-700'
              }`}
            >
              {replyResult.message}
            </div>
          )}
          <form onSubmit={handleReply} className="mt-4 space-y-3">
            <div>
              <label htmlFor="replyBody" className="block text-sm font-medium text-gray-700">
                Message
              </label>
              <textarea
                id="replyBody"
                value={replyBody}
                onChange={(e) => setReplyBody(e.target.value)}
                required
                rows={3}
                placeholder="Type your reply..."
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
            <button
              type="submit"
              disabled={replySending}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 transition-colors"
            >
              {replySending ? 'Sending...' : 'Send Reply'}
            </button>
          </form>
        </div>
      )}

      {/* Debug Section */}
      {hasDebugInfo && (
        <div className="rounded-lg bg-white shadow-sm">
          <button
            onClick={() => setDebugOpen(!debugOpen)}
            className="flex w-full items-center justify-between px-6 py-4 text-left transition-colors hover:bg-gray-50"
          >
            <span className="text-sm font-medium text-gray-700">Debug Information</span>
            <svg
              className={`h-5 w-5 text-gray-400 transition-transform ${debugOpen ? 'rotate-180' : ''}`}
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
            </svg>
          </button>
          {debugOpen && (
            <div className="border-t border-gray-100 px-6 py-4 space-y-3">
              {message.modem_response && (
                <div>
                  <span className="block text-xs font-medium uppercase text-gray-500 mb-1">
                    Modem Response
                  </span>
                  <pre className="rounded-md bg-gray-50 p-3 text-xs text-gray-700 overflow-x-auto">
                    {message.modem_response}
                  </pre>
                </div>
              )}
              {message.error_message && (
                <div>
                  <span className="block text-xs font-medium uppercase text-gray-500 mb-1">
                    Error Message
                  </span>
                  <pre className="rounded-md bg-red-50 p-3 text-xs text-red-700 overflow-x-auto">
                    {message.error_message}
                  </pre>
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
