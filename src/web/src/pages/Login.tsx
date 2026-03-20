import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '@/lib/auth';
import ThemeModeControl from '@/components/ThemeModeControl';

export default function Login() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login, isAuthenticated } = useAuth();
  const navigate = useNavigate();

  if (isAuthenticated) {
    navigate('/', { replace: true });
    return null;
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await login(username, password);
      navigate('/');
    } catch {
      setError('Invalid username or password');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="relative flex min-h-screen items-center justify-center bg-gray-100 px-4 dark:bg-[#002b36]">
      <div className="absolute right-4 top-4">
        <ThemeModeControl />
      </div>
      <div className="w-full max-w-sm">
        <h1 className="mb-8 text-center text-2xl font-bold text-gray-900 dark:text-[#fdf6e3]">
          SMS Gateway
        </h1>
        <form
          onSubmit={handleSubmit}
          className="rounded-lg bg-white p-8 shadow-md dark:bg-[#073642] dark:ring-1 dark:ring-[#586e75]"
        >
          <h2 className="mb-6 text-lg font-semibold text-gray-800 dark:text-[#eee8d5]">Sign in</h2>
          {error && (
            <div className="mb-4 rounded-md bg-red-50 p-3 text-sm text-red-700 dark:bg-[#3b1f23] dark:text-[#dc322f]">
              {error}
            </div>
          )}
          <div className="mb-4">
            <label
              htmlFor="username"
              className="mb-1 block text-sm font-medium text-gray-700 dark:text-[#93a1a1]"
            >
              Username
            </label>
            <input
              id="username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              autoFocus
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-[#586e75] dark:bg-[#002b36] dark:text-[#eee8d5] dark:focus:border-[#268bd2] dark:focus:ring-[#268bd2]"
            />
          </div>
          <div className="mb-6">
            <label
              htmlFor="password"
              className="mb-1 block text-sm font-medium text-gray-700 dark:text-[#93a1a1]"
            >
              Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-[#586e75] dark:bg-[#002b36] dark:text-[#eee8d5] dark:focus:border-[#268bd2] dark:focus:ring-[#268bd2]"
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 dark:bg-[#268bd2] dark:text-[#fdf6e3] dark:hover:bg-[#2aa5f5] dark:focus:ring-[#268bd2] dark:focus:ring-offset-[#073642]"
          >
            {loading ? 'Signing in...' : 'Sign in'}
          </button>
        </form>
      </div>
    </div>
  );
}
