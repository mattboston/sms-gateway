import { useState, useEffect, useCallback, type FormEvent } from 'react';
import { useAuth } from '@/lib/auth';
import api from '@/lib/api';

interface UserRecord {
  id: string;
  username: string;
  is_admin: boolean;
  created_at: string;
}

export default function Users() {
  const { user: currentUser } = useAuth();
  const [users, setUsers] = useState<UserRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  // Create form state
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [isAdmin, setIsAdmin] = useState(false);
  const [creating, setCreating] = useState(false);

  const isCurrentUserAdmin = currentUser?.is_admin ?? false;

  const fetchUsers = useCallback(async () => {
    try {
      const response = await api.get('/users');
      setUsers(response.data);
      setError('');
    } catch {
      setError('Failed to load users.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault();
    if (!username.trim() || !password.trim()) return;

    setCreating(true);
    setError('');
    setSuccess('');

    try {
      await api.post('/users', {
        username: username.trim(),
        password: password.trim(),
        is_admin: isAdmin,
      });
      setSuccess(`User "${username.trim()}" created successfully.`);
      setUsername('');
      setPassword('');
      setIsAdmin(false);
      await fetchUsers();
    } catch {
      setError('Failed to create user. Username may already exist.');
    } finally {
      setCreating(false);
    }
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

  if (!isCurrentUserAdmin) {
    return (
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Users</h1>
        <div className="mt-4 rounded-md bg-red-50 p-4 text-sm text-red-700">
          You do not have permission to view this page. Admin access is required.
        </div>
      </div>
    );
  }

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900">Users</h1>
      <p className="mt-1 text-sm text-gray-600">Manage user accounts.</p>

      {error && (
        <div className="mt-4 rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</div>
      )}
      {success && (
        <div className="mt-4 rounded-md bg-green-50 p-3 text-sm text-green-700">{success}</div>
      )}

      {/* Create user form */}
      <div className="mt-6 rounded-lg bg-white p-6 shadow-md">
        <h2 className="mb-4 text-lg font-semibold text-gray-800">Create New User</h2>
        <form onSubmit={handleCreate} className="space-y-4">
          <div className="flex flex-col gap-4 sm:flex-row">
            <div className="flex-1">
              <label htmlFor="newUsername" className="mb-1 block text-sm font-medium text-gray-700">
                Username
              </label>
              <input
                id="newUsername"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                placeholder="Enter username"
                className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
            <div className="flex-1">
              <label htmlFor="newPassword" className="mb-1 block text-sm font-medium text-gray-700">
                Password
              </label>
              <input
                id="newPassword"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                minLength={8}
                placeholder="Minimum 8 characters"
                className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <input
              id="isAdmin"
              type="checkbox"
              checked={isAdmin}
              onChange={(e) => setIsAdmin(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <label htmlFor="isAdmin" className="text-sm font-medium text-gray-700">
              Administrator
            </label>
          </div>
          <button
            type="submit"
            disabled={creating}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 transition-colors"
          >
            {creating ? 'Creating...' : 'Create User'}
          </button>
        </form>
      </div>

      {/* Users table */}
      <div className="mt-6 rounded-lg bg-white shadow-md overflow-hidden">
        {loading ? (
          <div className="p-6 text-center text-sm text-gray-500">Loading users...</div>
        ) : users.length === 0 ? (
          <div className="p-6 text-center text-sm text-gray-500">No users found.</div>
        ) : (
          <table className="w-full text-left text-sm">
            <thead className="border-b border-gray-200 bg-gray-50">
              <tr>
                <th className="px-6 py-3 font-medium text-gray-600">Username</th>
                <th className="px-6 py-3 font-medium text-gray-600">Role</th>
                <th className="px-6 py-3 font-medium text-gray-600">Created</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {users.map((u) => (
                <tr
                  key={u.id}
                  className={
                    u.id === currentUser?.id
                      ? 'bg-blue-50 hover:bg-blue-100'
                      : 'hover:bg-gray-50'
                  }
                >
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-gray-900">{u.username}</span>
                      {u.id === currentUser?.id && (
                        <span className="inline-flex items-center rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700">
                          You
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    {u.is_admin ? (
                      <span className="inline-flex items-center rounded-full bg-purple-100 px-2.5 py-0.5 text-xs font-medium text-purple-800">
                        Admin
                      </span>
                    ) : (
                      <span className="inline-flex items-center rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600">
                        User
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4 text-gray-600">{formatDate(u.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
