import { useState, useEffect, useCallback, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '@/lib/api';

interface ModemStatus {
  status: string;
}

interface ModemSignal {
  signal: number;
  quality: string;
}

interface Message {
  id: string;
  direction: 'inbound' | 'outbound';
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

function signalBars(quality: string): number {
  switch (quality) {
    case 'excellent': return 5;
    case 'good': return 4;
    case 'fair': return 3;
    case 'poor': return 2;
    case 'none': return 0;
    default: return 0;
  }
}

export default function Dashboard() {
  const navigate = useNavigate();
  const [modemStatus, setModemStatus] = useState<ModemStatus | null>(null);
  const [modemSignal, setModemSignal] = useState<ModemSignal | null>(null);
  const [recentMessages, setRecentMessages] = useState<Message[]>([]);
  const [totalSent, setTotalSent] = useState(0);
  const [totalReceived, setTotalReceived] = useState(0);
  const [pendingCount, setPendingCount] = useState(0);
  const [loading, setLoading] = useState(true);

  // Quick send form
  const [to, setTo] = useState('');
  const [body, setBody] = useState('');
  const [sending, setSending] = useState(false);
  const [sendResult, setSendResult] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  const fetchData = useCallback(async () => {
    try {
      const [statusRes, signalRes, inboxRes, outboxRes] = await Promise.allSettled([
        api.get<ModemStatus>('/modem/status'),
        api.get<ModemSignal>('/modem/signal'),
        api.get<Message[]>('/sms/inbox?all=true'),
        api.get<Message[]>('/sms/outbox'),
      ]);

      if (statusRes.status === 'fulfilled') setModemStatus(statusRes.value.data);
      else setModemStatus({ status: 'error' });

      if (signalRes.status === 'fulfilled') setModemSignal(signalRes.value.data);

      const inbox = inboxRes.status === 'fulfilled' ? inboxRes.value.data : [];
      const outbox = outboxRes.status === 'fulfilled' ? outboxRes.value.data : [];

      setTotalReceived(inbox.length);
      setTotalSent(outbox.filter((m) => m.status === 'sent').length);
      setPendingCount(outbox.filter((m) => m.status === 'pending' || m.status === 'sending').length);

      const combined = [...inbox, ...outbox]
        .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
        .slice(0, 10);
      setRecentMessages(combined);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
    const interval = setInterval(async () => {
      try {
        const [statusRes, signalRes] = await Promise.allSettled([
          api.get<ModemStatus>('/modem/status'),
          api.get<ModemSignal>('/modem/signal'),
        ]);
        if (statusRes.status === 'fulfilled') setModemStatus(statusRes.value.data);
        else setModemStatus({ status: 'error' });
        if (signalRes.status === 'fulfilled') setModemSignal(signalRes.value.data);
      } catch {
        // ignore
      }
    }, 30000);
    return () => clearInterval(interval);
  }, [fetchData]);

  const handleQuickSend = async (e: FormEvent) => {
    e.preventDefault();
    setSending(true);
    setSendResult(null);
    try {
      const res = await api.post('/sms/send', { to, body });
      if (res.data.status === 'sent') {
        setSendResult({ type: 'success', message: 'Message sent successfully.' });
        setTo('');
        setBody('');
        fetchData();
      } else {
        setSendResult({ type: 'error', message: res.data.message || 'Failed to send message.' });
      }
    } catch {
      setSendResult({ type: 'error', message: 'Failed to send message.' });
    } finally {
      setSending(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <p className="text-gray-500">Loading dashboard...</p>
      </div>
    );
  }

  const modemOk = modemStatus?.status === 'ok';
  const bars = modemSignal ? signalBars(modemSignal.quality) : 0;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {/* Modem Status */}
        <div className="rounded-lg bg-white p-5 shadow-sm">
          <div className="text-sm font-medium text-gray-500">Modem Status</div>
          <div className="mt-2 flex items-center gap-2">
            <span
              className={`inline-block h-3 w-3 rounded-full ${modemOk ? 'bg-green-500' : 'bg-red-500'}`}
            />
            <span className={`text-lg font-semibold ${modemOk ? 'text-green-700' : 'text-red-700'}`}>
              {modemOk ? 'Online' : 'Offline'}
            </span>
          </div>
        </div>

        {/* Signal Strength */}
        <div className="rounded-lg bg-white p-5 shadow-sm">
          <div className="text-sm font-medium text-gray-500">Signal Strength</div>
          <div className="mt-2 flex items-end gap-1">
            {[1, 2, 3, 4, 5].map((level) => (
              <div
                key={level}
                className={`w-2 rounded-sm ${bars >= level ? 'bg-green-500' : 'bg-gray-200'}`}
                style={{ height: `${level * 5 + 4}px` }}
              />
            ))}
            <span className="ml-2 text-sm text-gray-600 capitalize">
              {modemSignal ? modemSignal.quality : 'unknown'}
              {modemSignal && modemSignal.quality !== 'unknown' ? ` (${modemSignal.signal})` : ''}
            </span>
          </div>
        </div>

        {/* Total Sent */}
        <div className="rounded-lg bg-white p-5 shadow-sm">
          <div className="text-sm font-medium text-gray-500">Total Sent</div>
          <div className="mt-2 text-2xl font-bold text-gray-900">{totalSent}</div>
        </div>

        {/* Total Received */}
        <div className="rounded-lg bg-white p-5 shadow-sm">
          <div className="text-sm font-medium text-gray-500">Total Received</div>
          <div className="mt-2 text-2xl font-bold text-gray-900">{totalReceived}</div>
        </div>
      </div>

      {/* Pending Messages Banner */}
      {pendingCount > 0 && (
        <div className="rounded-lg bg-yellow-50 border border-yellow-200 px-4 py-3 text-sm text-yellow-800">
          {pendingCount} message{pendingCount !== 1 ? 's' : ''} pending delivery.
        </div>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Quick Send */}
        <div className="rounded-lg bg-white p-5 shadow-sm">
          <h2 className="text-lg font-semibold text-gray-800">Quick Send</h2>
          {sendResult && (
            <div
              className={`mt-3 rounded-md p-3 text-sm ${
                sendResult.type === 'success'
                  ? 'bg-green-50 text-green-700'
                  : 'bg-red-50 text-red-700'
              }`}
            >
              {sendResult.message}
            </div>
          )}
          <form onSubmit={handleQuickSend} className="mt-4 space-y-3">
            <div>
              <label htmlFor="quickTo" className="block text-sm font-medium text-gray-700">
                Phone Number
              </label>
              <input
                id="quickTo"
                type="tel"
                value={to}
                onChange={(e) => setTo(e.target.value)}
                required
                placeholder="+1234567890"
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
            <div>
              <label htmlFor="quickBody" className="block text-sm font-medium text-gray-700">
                Message
              </label>
              <textarea
                id="quickBody"
                value={body}
                onChange={(e) => setBody(e.target.value)}
                required
                rows={3}
                placeholder="Type your message..."
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
            <button
              type="submit"
              disabled={sending}
              className="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 transition-colors"
            >
              {sending ? 'Sending...' : 'Send SMS'}
            </button>
          </form>
        </div>

        {/* Recent Messages */}
        <div className="rounded-lg bg-white p-5 shadow-sm lg:col-span-2">
          <h2 className="text-lg font-semibold text-gray-800">Recent Messages</h2>
          {recentMessages.length === 0 ? (
            <p className="mt-4 text-sm text-gray-500">No messages yet.</p>
          ) : (
            <div className="mt-4 divide-y divide-gray-100">
              {recentMessages.map((msg) => (
                <div
                  key={msg.id}
                  className="flex items-start justify-between py-3 cursor-pointer hover:bg-gray-50 -mx-2 px-2 rounded transition-colors"
                  onClick={() => navigate(`/messages/${msg.id}`)}
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span
                        className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                          msg.direction === 'inbound'
                            ? 'bg-blue-100 text-blue-700'
                            : 'bg-gray-100 text-gray-700'
                        }`}
                      >
                        {msg.direction === 'inbound' ? 'IN' : 'OUT'}
                      </span>
                      <span className={`text-sm ${msg.direction === 'inbound' && msg.status === 'received' ? 'font-bold text-gray-900' : 'font-medium text-gray-900'}`}>{msg.phone_number}</span>
                      {msg.direction === 'inbound' && (
                        <span
                          className={`inline-flex items-center rounded-full px-1.5 py-0.5 text-xs font-medium ${
                            msg.status === 'received'
                              ? 'bg-blue-100 text-blue-800'
                              : 'bg-gray-100 text-gray-600'
                          }`}
                        >
                          {msg.status === 'received' ? 'Unread' : 'Read'}
                        </span>
                      )}
                    </div>
                    <p className={`mt-0.5 truncate text-sm ${msg.direction === 'inbound' && msg.status === 'received' ? 'font-semibold text-gray-900' : 'text-gray-600'}`}>{msg.body}</p>
                  </div>
                  <span className="ml-4 shrink-0 text-xs text-gray-400">
                    {formatRelativeTime(msg.created_at)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
