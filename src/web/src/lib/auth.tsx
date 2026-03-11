import { createContext, useContext, useState, useCallback, type ReactNode } from 'react';
import api from '@/lib/api';

interface User {
  id: string;
  username: string;
  is_admin: boolean;
  must_change_password: boolean;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
  clearMustChangePassword: () => void;
  isAuthenticated: boolean;
  mustChangePassword: boolean;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(() => {
    const stored = localStorage.getItem('user');
    return stored ? JSON.parse(stored) : null;
  });

  const [token, setToken] = useState<string | null>(() => {
    return localStorage.getItem('token');
  });

  const login = useCallback(async (username: string, password: string) => {
    const response = await api.post('/auth/login', { username, password });
    const { token: newToken, user: newUser } = response.data;
    localStorage.setItem('token', newToken);
    localStorage.setItem('user', JSON.stringify(newUser));
    setToken(newToken);
    setUser(newUser);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    setToken(null);
    setUser(null);
  }, []);

  const clearMustChangePassword = useCallback(() => {
    if (user) {
      const updated = { ...user, must_change_password: false };
      localStorage.setItem('user', JSON.stringify(updated));
      setUser(updated);
    }
  }, [user]);

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        login,
        logout,
        clearMustChangePassword,
        isAuthenticated: !!token,
        mustChangePassword: user?.must_change_password ?? false,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
