import { useState, useEffect, type FormEvent } from 'react';
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

const SMS_CHAR_LIMIT = 160;

function statusBadge(status: string) {
  const styles: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-800 dark:bg-[#3b3200] dark:text-[#b58900]',
    sending: 'bg-blue-100 text-blue-800 dark:bg-[#1f3e52] dark:text-[#268bd2]',
    sent: 'bg-green-100 text-green-800 dark:bg-[#213a25] dark:text-[#859900]',
    failed: 'bg-red-100 text-red-800 dark:bg-[#3b1f23] dark:text-[#dc322f]',
  };
  return styles[status] ?? 'bg-gray-100 text-gray-800 dark:bg-[#586e75] dark:text-[#eee8d5]';
}

export default function SendSMS() {
  const navigate = useNavigate();
  const [to, setTo] = useState('');
  const [body, setBody] = useState('');
  const [sending, setSending] = useState(false);
  const [result, setResult] = useState<{ type: 'success' | 'error'; message: string } | null>(null);
  const [sentMessages, setSentMessages] = useState<Message[]>([]);
  const [loadingMessages, setLoadingMessages] = useState(true);

  const fetchSentMessages = async () => {
    try {
      const res = await api.get<Message[]>('/sms/outbox');
      setSentMessages(res.data.slice(0, 20));
    } catch {
      // ignore
    } finally {
      setLoadingMessages(false);
    }
  };

  useEffect(() => {
    fetchSentMessages();
  }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setSending(true);
    setResult(null);
    try {
      const res = await api.post('/sms/send', { to, body });
      if (res.data.status === 'sent') {
        setResult({ type: 'success', message: 'Message sent successfully.' });
        setTo('');
        setBody('');
        fetchSentMessages();
      } else {
        setResult({ type: 'error', message: res.data.message || 'Failed to send message.' });
      }
    } catch {
      setResult({ type: 'error', message: 'Failed to send message.' });
    } finally {
      setSending(false);
    }
  };

  const charCount = body.length;
  const overLimit = charCount > SMS_CHAR_LIMIT;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-gray-900 dark:text-[#fdf6e3]">Send SMS</h1>

      {/* Send Form */}
      <div className="max-w-2xl rounded-lg bg-white p-6 shadow-sm dark:bg-[#073642] dark:ring-1 dark:ring-[#586e75]">
        {result && (
          <div
            className={`mb-4 rounded-md p-3 text-sm ${
              result.type === 'success' ? 'bg-green-50 text-green-700 dark:bg-[#213a25] dark:text-[#859900]' : 'bg-red-50 text-red-700 dark:bg-[#3b1f23] dark:text-[#dc322f]'
            }`}
          >
            {result.message}
          </div>
        )}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="to" className="block text-sm font-medium text-gray-700 dark:text-[#93a1a1]">
              Phone Number
            </label>
            <input
              id="to"
              type="tel"
              value={to}
              onChange={(e) => setTo(e.target.value)}
              required
              placeholder="+1234567890"
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-[#586e75] dark:bg-[#002b36] dark:text-[#eee8d5] dark:focus:border-[#268bd2] dark:focus:ring-[#268bd2]"
            />
          </div>
          <div>
            <label htmlFor="body" className="block text-sm font-medium text-gray-700 dark:text-[#93a1a1]">
              Message
            </label>
            <textarea
              id="body"
              value={body}
              onChange={(e) => setBody(e.target.value)}
              required
              rows={4}
              placeholder="Type your message..."
              className={`mt-1 block w-full rounded-md border px-3 py-2 text-sm focus:outline-none focus:ring-1 ${
                overLimit
                  ? 'border-red-300 focus:border-red-500 focus:ring-red-500 dark:border-[#dc322f] dark:bg-[#002b36] dark:text-[#eee8d5] dark:focus:border-[#dc322f] dark:focus:ring-[#dc322f]'
                  : 'border-gray-300 focus:border-blue-500 focus:ring-blue-500 dark:border-[#586e75] dark:bg-[#002b36] dark:text-[#eee8d5] dark:focus:border-[#268bd2] dark:focus:ring-[#268bd2]'
              }`}
            />
            <div className="mt-1 flex justify-between text-xs">
              <span className={overLimit ? 'text-red-600 font-medium dark:text-[#dc322f]' : 'text-gray-400 dark:text-[#93a1a1]'}>
                {charCount}/{SMS_CHAR_LIMIT} characters
                {overLimit && ' - message may be split into multiple SMS'}
              </span>
            </div>
          </div>
          <button
            type="submit"
            disabled={sending || !to || !body}
            className="rounded-md bg-blue-600 px-6 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 dark:bg-[#268bd2] dark:text-[#fdf6e3] dark:hover:bg-[#2aa5f5] dark:focus:ring-[#268bd2] dark:focus:ring-offset-[#073642]"
          >
            {sending ? 'Sending...' : 'Send Message'}
          </button>
        </form>
      </div>

      {/* Recent Sent Messages */}
      <div className="rounded-lg bg-white p-6 shadow-sm dark:bg-[#073642] dark:ring-1 dark:ring-[#586e75]">
        <h2 className="text-lg font-semibold text-gray-800 dark:text-[#eee8d5]">Recent Sent Messages</h2>
        {loadingMessages ? (
          <p className="mt-4 text-sm text-gray-500 dark:text-[#93a1a1]">Loading...</p>
        ) : sentMessages.length === 0 ? (
          <p className="mt-4 text-sm text-gray-500 dark:text-[#93a1a1]">No sent messages yet.</p>
        ) : (
          <div className="mt-4 overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 text-left text-xs font-medium uppercase text-gray-500 dark:border-[#586e75] dark:text-[#93a1a1]">
                  <th className="pb-2 pr-4">To</th>
                  <th className="pb-2 pr-4">Message</th>
                  <th className="pb-2 pr-4">Status</th>
                  <th className="pb-2">Time</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-[#586e75]">
                {sentMessages.map((msg) => (
                  <tr
                    key={msg.id}
                    className="cursor-pointer transition-colors hover:bg-gray-50 dark:hover:bg-[#0a4452]"
                    onClick={() => navigate(`/messages/${msg.id}`)}
                  >
                    <td className="py-3 pr-4 font-medium text-gray-900 dark:text-[#eee8d5]">{msg.phone_number}</td>
                    <td className="py-3 pr-4 text-gray-600 max-w-xs truncate dark:text-[#93a1a1]">{msg.body}</td>
                    <td className="py-3 pr-4">
                      <span
                        className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${statusBadge(msg.status)}`}
                      >
                        {msg.status}
                      </span>
                    </td>
                    <td className="py-3 text-gray-400 dark:text-[#93a1a1]">{formatRelativeTime(msg.created_at)}</td>
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
