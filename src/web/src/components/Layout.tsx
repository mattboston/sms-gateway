import { useState } from 'react';
import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import { useAuth } from '@/lib/auth';
import ThemeModeControl from '@/components/ThemeModeControl';

const navItems = [
  { to: '/', label: 'Dashboard' },
  { to: '/send', label: 'Send SMS' },
  { to: '/inbox', label: 'Inbox' },
  { to: '/outbox', label: 'Outbox' },
  { to: '/apikeys', label: 'API Keys' },
  { to: '/users', label: 'Users' },
  { to: '/modem', label: 'Modem Test' },
];

export default function Layout() {
  const { logout, user } = useAuth();
  const navigate = useNavigate();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div className="flex h-screen bg-gray-100 dark:bg-[#002b36]">
      {/* Mobile overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-20 bg-black/50 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={`fixed inset-y-0 left-0 z-30 w-64 transform bg-gray-900 text-white transition-transform duration-200 lg:static lg:translate-x-0 dark:bg-[#073642] dark:text-[#93a1a1] ${
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="flex h-16 items-center justify-between border-b border-gray-700 px-6 dark:border-[#586e75]">
          <span className="text-lg font-semibold">SMS Gateway</span>
          <button
            className="lg:hidden text-gray-400 hover:text-white dark:text-[#93a1a1] dark:hover:text-[#fdf6e3]"
            onClick={() => setSidebarOpen(false)}
          >
            <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>
        <nav className="mt-4 px-3 space-y-1">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              onClick={() => setSidebarOpen(false)}
              className={({ isActive }) =>
                `block rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-gray-800 text-white dark:bg-[#0a4452] dark:text-[#fdf6e3]'
                    : 'text-gray-300 hover:bg-gray-800 hover:text-white dark:text-[#93a1a1] dark:hover:bg-[#0a4452] dark:hover:text-[#fdf6e3]'
                }`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>

      {/* Main content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Header */}
        <header className="flex h-16 items-center justify-between border-b border-gray-200 bg-white px-6 dark:border-[#586e75] dark:bg-[#002b36]">
          <button
            className="lg:hidden text-gray-600 hover:text-gray-900 dark:text-[#93a1a1] dark:hover:text-[#fdf6e3]"
            onClick={() => setSidebarOpen(true)}
          >
            <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M4 6h16M4 12h16M4 18h16"
              />
            </svg>
          </button>
          <div className="flex-1 lg:ml-0" />
          <div className="flex items-center gap-4">
            <ThemeModeControl />
            {user && <span className="text-sm text-gray-600 dark:text-[#93a1a1]">{user.username}</span>}
            <button
              onClick={handleLogout}
              className="rounded-md bg-gray-100 px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 dark:bg-[#073642] dark:text-[#93a1a1] dark:hover:bg-[#0a4452]"
            >
              Logout
            </button>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-y-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
