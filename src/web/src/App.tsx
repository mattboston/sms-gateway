import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from '@/lib/auth';
import Layout from '@/components/Layout';
import Dashboard from '@/pages/Dashboard';
import Login from '@/pages/Login';
import ChangePassword from '@/pages/ChangePassword';
import SendSMS from '@/pages/SendSMS';
import Inbox from '@/pages/Inbox';
import Outbox from '@/pages/Outbox';
import MessageDetail from '@/pages/MessageDetail';
import APIKeys from '@/pages/APIKeys';
import Users from '@/pages/Users';
import ModemTest from '@/pages/ModemTest';
import type { ReactNode } from 'react';

function ProtectedRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated, mustChangePassword } = useAuth();
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  if (mustChangePassword) {
    return <Navigate to="/change-password" replace />;
  }
  return <>{children}</>;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/change-password" element={<ChangePassword />} />
      <Route
        element={
          <ProtectedRoute>
            <Layout />
          </ProtectedRoute>
        }
      >
        <Route path="/" element={<Dashboard />} />
        <Route path="/send" element={<SendSMS />} />
        <Route path="/inbox" element={<Inbox />} />
        <Route path="/outbox" element={<Outbox />} />
        <Route path="/messages/:id" element={<MessageDetail />} />
        <Route path="/apikeys" element={<APIKeys />} />
        <Route path="/users" element={<Users />} />
        <Route path="/modem" element={<ModemTest />} />
      </Route>
    </Routes>
  );
}
