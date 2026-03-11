import { useState, useEffect, useCallback, type FormEvent } from 'react';
import { useAuth } from '@/lib/auth';
import api from '@/lib/api';

interface ModemStatus {
  status: string;
}

interface ModemSignal {
  signal: number;
  quality: string;
}

interface CommandEntry {
  command: string;
  response: string;
  timestamp: Date;
}

function SignalBars({ strength }: { strength: number }) {
  // Normalize strength to 0-4 bars (signal_strength is typically 0-31 for GSM)
  const bars = Math.min(4, Math.max(0, Math.round((strength / 31) * 4)));

  return (
    <div className="flex items-end gap-1" title={`Signal: ${strength}/31`}>
      {[1, 2, 3, 4].map((level) => (
        <div
          key={level}
          className={`w-2 rounded-sm transition-colors ${
            level <= bars ? 'bg-green-500' : 'bg-gray-200'
          }`}
          style={{ height: `${level * 6 + 4}px` }}
        />
      ))}
    </div>
  );
}

export default function ModemTest() {
  const { user } = useAuth();
  const isAdmin = user?.is_admin ?? false;

  const [status, setStatus] = useState<ModemStatus | null>(null);
  const [signal, setSignal] = useState<ModemSignal | null>(null);
  const [statusLoading, setStatusLoading] = useState(true);
  const [signalLoading, setSignalLoading] = useState(true);
  const [statusError, setStatusError] = useState('');
  const [signalError, setSignalError] = useState('');

  // AT command state
  const [command, setCommand] = useState('');
  const [sending, setSending] = useState(false);
  const [commandError, setCommandError] = useState('');
  const [history, setHistory] = useState<CommandEntry[]>([]);

  const fetchStatus = useCallback(async () => {
    setStatusLoading(true);
    setStatusError('');
    try {
      const response = await api.get('/modem/status');
      setStatus(response.data);
    } catch {
      setStatusError('Failed to fetch modem status.');
    } finally {
      setStatusLoading(false);
    }
  }, []);

  const fetchSignal = useCallback(async () => {
    setSignalLoading(true);
    setSignalError('');
    try {
      const response = await api.get('/modem/signal');
      setSignal(response.data);
    } catch {
      setSignalError('Failed to fetch signal information.');
    } finally {
      setSignalLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStatus();
    fetchSignal();
  }, [fetchStatus, fetchSignal]);

  const handleRefresh = () => {
    fetchStatus();
    fetchSignal();
  };

  const handleSendCommand = async (e: FormEvent) => {
    e.preventDefault();
    if (!command.trim()) return;

    setSending(true);
    setCommandError('');
    const cmd = command.trim();

    try {
      const response = await api.post('/modem/at', { command: cmd });
      setHistory((prev) => [
        { command: cmd, response: response.data.response ?? JSON.stringify(response.data), timestamp: new Date() },
        ...prev,
      ]);
      setCommand('');
    } catch (err: unknown) {
      const message =
        err instanceof Error
          ? err.message
          : typeof err === 'object' && err !== null && 'response' in err
            ? String((err as { response?: { data?: { error?: string } } }).response?.data?.error ?? 'Command failed.')
            : 'Command failed.';
      setCommandError(message);
      setHistory((prev) => [
        { command: cmd, response: `ERROR: ${message}`, timestamp: new Date() },
        ...prev,
      ]);
    } finally {
      setSending(false);
    }
  };

  const formatTime = (date: Date) => {
    return date.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  };

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Modem Test</h1>
          <p className="mt-1 text-sm text-gray-600">Test modem connectivity and AT commands.</p>
        </div>
        <button
          onClick={handleRefresh}
          disabled={statusLoading || signalLoading}
          className="rounded-md bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 transition-colors"
        >
          {statusLoading || signalLoading ? 'Refreshing...' : 'Refresh'}
        </button>
      </div>

      {/* Status and Signal cards */}
      <div className="mt-6 grid gap-6 sm:grid-cols-2">
        {/* Modem Status */}
        <div className="rounded-lg bg-white p-6 shadow-md">
          <h2 className="mb-4 text-lg font-semibold text-gray-800">Modem Status</h2>
          {statusLoading ? (
            <p className="text-sm text-gray-500">Loading...</p>
          ) : statusError ? (
            <p className="text-sm text-red-600">{statusError}</p>
          ) : status ? (
            <div className="space-y-3">
              <div className="flex items-center gap-3">
                <div
                  className={`h-3 w-3 rounded-full ${
                    status.status === 'ok' ? 'bg-green-500' : 'bg-red-500'
                  }`}
                />
                <span className="text-sm font-medium text-gray-900">
                  {status.status === 'ok' ? 'Connected' : 'Disconnected'}
                </span>
              </div>
            </div>
          ) : (
            <p className="text-sm text-gray-500">No status data available.</p>
          )}
        </div>

        {/* Signal Strength */}
        <div className="rounded-lg bg-white p-6 shadow-md">
          <h2 className="mb-4 text-lg font-semibold text-gray-800">Signal Strength</h2>
          {signalLoading ? (
            <p className="text-sm text-gray-500">Loading...</p>
          ) : signalError ? (
            <p className="text-sm text-red-600">{signalError}</p>
          ) : signal ? (
            <div className="space-y-3">
              <div className="flex items-center gap-4">
                <SignalBars strength={signal.signal} />
                <span className="text-2xl font-bold text-gray-900">
                  {signal.signal}
                </span>
                <span className="text-sm text-gray-500">/ 31</span>
              </div>
              <div>
                <span
                  className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                    signal.quality === 'excellent' || signal.quality === 'good'
                      ? 'bg-green-100 text-green-800'
                      : signal.quality === 'fair'
                        ? 'bg-yellow-100 text-yellow-800'
                        : 'bg-red-100 text-red-800'
                  }`}
                >
                  {signal.quality}
                </span>
              </div>
            </div>
          ) : (
            <p className="text-sm text-gray-500">No signal data available.</p>
          )}
        </div>
      </div>

      {/* AT Command Section - Admin only */}
      {isAdmin && (
        <div className="mt-6 rounded-lg bg-white p-6 shadow-md">
          <h2 className="mb-4 text-lg font-semibold text-gray-800">AT Command</h2>
          <form onSubmit={handleSendCommand} className="flex items-end gap-3">
            <div className="flex-1">
              <label htmlFor="atCommand" className="mb-1 block text-sm font-medium text-gray-700">
                Command
              </label>
              <input
                id="atCommand"
                type="text"
                value={command}
                onChange={(e) => setCommand(e.target.value)}
                required
                placeholder="e.g. AT+CSQ"
                className="w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
            <button
              type="submit"
              disabled={sending}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 transition-colors"
            >
              {sending ? 'Sending...' : 'Send'}
            </button>
          </form>

          {commandError && (
            <div className="mt-3 rounded-md bg-red-50 p-3 text-sm text-red-700">
              {commandError}
            </div>
          )}

          {/* Command History */}
          {history.length > 0 && (
            <div className="mt-6">
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-sm font-semibold text-gray-700">Command History</h3>
                <button
                  onClick={() => setHistory([])}
                  className="text-xs text-gray-500 hover:text-gray-700"
                >
                  Clear
                </button>
              </div>
              <div className="space-y-3">
                {history.map((entry, index) => (
                  <div
                    key={index}
                    className="rounded-md border border-gray-200 bg-gray-50 p-3"
                  >
                    <div className="mb-1 flex items-center justify-between">
                      <code className="text-sm font-semibold text-blue-700">{entry.command}</code>
                      <span className="text-xs text-gray-400">{formatTime(entry.timestamp)}</span>
                    </div>
                    <pre className="whitespace-pre-wrap rounded bg-gray-900 p-3 font-mono text-xs text-green-400 overflow-x-auto">
                      {entry.response}
                    </pre>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
