import { useState, useEffect, useCallback, type FormEvent } from 'react';
import api from '@/lib/api';

interface APIKey {
  id: string;
  label: string;
  key: string;
  active: boolean;
  created_at: string;
}

export default function APIKeys() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [label, setLabel] = useState('');
  const [creating, setCreating] = useState(false);
  const [newKey, setNewKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [confirmId, setConfirmId] = useState<string | null>(null);
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
  const [revealedKeys, setRevealedKeys] = useState<Set<string>>(new Set());

  const fetchKeys = useCallback(async () => {
    try {
      const response = await api.get('/apikeys');
      setKeys(response.data);
      setError('');
    } catch {
      setError('Failed to load API keys.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchKeys();
  }, [fetchKeys]);

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault();
    if (!label.trim()) return;
    setCreating(true);
    setError('');
    try {
      const response = await api.post('/apikeys', { label: label.trim() });
      setNewKey(response.data.key);
      setLabel('');
      await fetchKeys();
    } catch {
      setError('Failed to create API key.');
    } finally {
      setCreating(false);
    }
  };

  const handleDeactivate = async (id: string) => {
    try {
      await api.delete(`/apikeys/${id}`);
      setConfirmId(null);
      await fetchKeys();
    } catch {
      setError('Failed to deactivate API key.');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await api.delete(`/apikeys/${id}/delete`);
      setDeleteConfirmId(null);
      await fetchKeys();
    } catch {
      setError('Failed to delete API key.');
    }
  };

  const copyToClipboard = async (text: string) => {
    try {
      if (navigator.clipboard) {
        await navigator.clipboard.writeText(text);
      } else {
        // Fallback for non-secure contexts (HTTP)
        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.opacity = '0';
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
      }
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setError('Failed to copy to clipboard.');
    }
  };

  const toggleReveal = (id: string) => {
    setRevealedKeys((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const maskKey = (key: string) => {
    if (key.length <= 8) return '********';
    return key.slice(0, 4) + '********' + key.slice(-4);
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900">API Keys</h1>
      <p className="mt-1 text-sm text-gray-600">Manage API keys for programmatic access.</p>

      {error && (
        <div className="mt-4 rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</div>
      )}

      {/* New key reveal banner */}
      {newKey && (
        <div className="mt-4 rounded-md border border-amber-300 bg-amber-50 p-4">
          <p className="mb-2 text-sm font-semibold text-amber-800">
            Your new API key has been created. Copy it now -- it will not be shown again.
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded bg-white px-3 py-2 font-mono text-sm text-gray-900 border border-amber-200">
              {newKey}
            </code>
            <button
              onClick={() => copyToClipboard(newKey)}
              className="rounded-md bg-amber-600 px-3 py-2 text-sm font-medium text-white hover:bg-amber-700 transition-colors"
            >
              {copied ? 'Copied!' : 'Copy'}
            </button>
          </div>
          <button
            onClick={() => setNewKey(null)}
            className="mt-2 text-sm text-amber-700 underline hover:text-amber-900"
          >
            Dismiss
          </button>
        </div>
      )}

      {/* Create form */}
      <div className="mt-6 rounded-lg bg-white p-6 shadow-md">
        <h2 className="mb-4 text-lg font-semibold text-gray-800">Create New API Key</h2>
        <form onSubmit={handleCreate} className="flex items-end gap-3">
          <div className="flex-1">
            <label htmlFor="keyLabel" className="mb-1 block text-sm font-medium text-gray-700">
              Label
            </label>
            <input
              id="keyLabel"
              type="text"
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              required
              placeholder="e.g. Production Server"
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
          <button
            type="submit"
            disabled={creating}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 transition-colors"
          >
            {creating ? 'Creating...' : 'Create Key'}
          </button>
        </form>
      </div>

      {/* Keys table */}
      <div className="mt-6 rounded-lg bg-white shadow-md overflow-hidden">
        {loading ? (
          <div className="p-6 text-center text-sm text-gray-500">Loading API keys...</div>
        ) : keys.length === 0 ? (
          <div className="p-6 text-center text-sm text-gray-500">No API keys found.</div>
        ) : (
          <table className="w-full text-left text-sm">
            <thead className="border-b border-gray-200 bg-gray-50">
              <tr>
                <th className="px-6 py-3 font-medium text-gray-600">Label</th>
                <th className="px-6 py-3 font-medium text-gray-600">Key</th>
                <th className="px-6 py-3 font-medium text-gray-600">Status</th>
                <th className="px-6 py-3 font-medium text-gray-600">Created</th>
                <th className="px-6 py-3 font-medium text-gray-600">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {keys.map((k) => (
                <tr key={k.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 font-medium text-gray-900">{k.label}</td>
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-2">
                      <code className="font-mono text-xs text-gray-600">
                        {revealedKeys.has(k.id) ? k.key : maskKey(k.key)}
                      </code>
                      <button
                        onClick={() => toggleReveal(k.id)}
                        className="text-xs text-blue-600 hover:text-blue-800"
                        title={revealedKeys.has(k.id) ? 'Hide' : 'Reveal'}
                      >
                        {revealedKeys.has(k.id) ? 'Hide' : 'Show'}
                      </button>
                      <button
                        onClick={() => copyToClipboard(k.key)}
                        className="text-xs text-blue-600 hover:text-blue-800"
                        title="Copy to clipboard"
                      >
                        Copy
                      </button>
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    {k.active ? (
                      <span className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800">
                        Active
                      </span>
                    ) : (
                      <span className="inline-flex items-center rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600">
                        Inactive
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4 text-gray-600">{formatDate(k.created_at)}</td>
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-2">
                      {k.active && (
                        <>
                          {confirmId === k.id ? (
                            <>
                              <span className="text-xs text-gray-600">Deactivate?</span>
                              <button
                                onClick={() => handleDeactivate(k.id)}
                                className="rounded bg-red-600 px-2 py-1 text-xs font-medium text-white hover:bg-red-700 transition-colors"
                              >
                                Yes
                              </button>
                              <button
                                onClick={() => setConfirmId(null)}
                                className="rounded bg-gray-200 px-2 py-1 text-xs font-medium text-gray-700 hover:bg-gray-300 transition-colors"
                              >
                                Cancel
                              </button>
                            </>
                          ) : (
                            <button
                              onClick={() => { setConfirmId(k.id); setDeleteConfirmId(null); }}
                              className="rounded bg-red-50 px-2.5 py-1 text-xs font-medium text-red-700 hover:bg-red-100 transition-colors"
                            >
                              Deactivate
                            </button>
                          )}
                        </>
                      )}
                      {deleteConfirmId === k.id ? (
                        <>
                          <span className="text-xs text-gray-600">Delete permanently?</span>
                          <button
                            onClick={() => handleDelete(k.id)}
                            className="rounded bg-red-600 px-2 py-1 text-xs font-medium text-white hover:bg-red-700 transition-colors"
                          >
                            Yes
                          </button>
                          <button
                            onClick={() => setDeleteConfirmId(null)}
                            className="rounded bg-gray-200 px-2 py-1 text-xs font-medium text-gray-700 hover:bg-gray-300 transition-colors"
                          >
                            Cancel
                          </button>
                        </>
                      ) : (
                        <button
                          onClick={() => { setDeleteConfirmId(k.id); setConfirmId(null); }}
                          className="rounded bg-red-50 px-2.5 py-1 text-xs font-medium text-red-700 hover:bg-red-100 transition-colors"
                        >
                          Delete
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
