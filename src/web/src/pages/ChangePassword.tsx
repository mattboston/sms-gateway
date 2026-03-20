import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '@/lib/auth';
import api from '@/lib/api';
import ThemeModeControl from '@/components/ThemeModeControl';

export default function ChangePassword() {
  const { mustChangePassword, clearMustChangePassword } = useAuth();
  const navigate = useNavigate();
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');

    if (newPassword !== confirmPassword) {
      setError('New passwords do not match.');
      return;
    }

    if (newPassword.length < 8) {
      setError('New password must be at least 8 characters.');
      return;
    }

    setLoading(true);
    try {
      await api.post('/auth/change-password', {
        current_password: currentPassword,
        new_password: newPassword,
      });
      clearMustChangePassword();
      navigate('/');
    } catch {
      setError('Failed to change password. Check your current password.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="relative flex min-h-screen items-center justify-center bg-gray-100 dark:bg-[#002b36]">
      <div className="absolute right-4 top-4">
        <ThemeModeControl />
      </div>
      <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-md dark:bg-[#073642] dark:ring-1 dark:ring-[#586e75]">
        <h1 className="mb-2 text-2xl font-bold text-gray-900 dark:text-[#fdf6e3]">Change Password</h1>
        {mustChangePassword && (
          <p className="mb-4 text-sm text-amber-600 dark:text-[#b58900]">
            You must change your password before continuing.
          </p>
        )}
        {error && (
          <div className="mb-4 rounded bg-red-50 p-3 text-sm text-red-600 dark:bg-[#3b1f23] dark:text-[#dc322f]">
            {error}
          </div>
        )}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label
              htmlFor="currentPassword"
              className="block text-sm font-medium text-gray-700 dark:text-[#93a1a1]"
            >
              Current Password
            </label>
            <input
              id="currentPassword"
              type="password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              required
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-[#586e75] dark:bg-[#002b36] dark:text-[#eee8d5] dark:focus:border-[#268bd2] dark:focus:ring-[#268bd2]"
            />
          </div>
          <div>
            <label
              htmlFor="newPassword"
              className="block text-sm font-medium text-gray-700 dark:text-[#93a1a1]"
            >
              New Password
            </label>
            <input
              id="newPassword"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
              minLength={8}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-[#586e75] dark:bg-[#002b36] dark:text-[#eee8d5] dark:focus:border-[#268bd2] dark:focus:ring-[#268bd2]"
            />
          </div>
          <div>
            <label
              htmlFor="confirmPassword"
              className="block text-sm font-medium text-gray-700 dark:text-[#93a1a1]"
            >
              Confirm New Password
            </label>
            <input
              id="confirmPassword"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
              minLength={8}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-[#586e75] dark:bg-[#002b36] dark:text-[#eee8d5] dark:focus:border-[#268bd2] dark:focus:ring-[#268bd2]"
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-md bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 dark:bg-[#268bd2] dark:text-[#fdf6e3] dark:hover:bg-[#2aa5f5] dark:focus:ring-[#268bd2] dark:focus:ring-offset-[#073642]"
          >
            {loading ? 'Changing...' : 'Change Password'}
          </button>
        </form>
      </div>
    </div>
  );
}
