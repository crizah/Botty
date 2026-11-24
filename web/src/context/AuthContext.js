import { createContext, useCallback, useContext, useEffect, useState } from 'react';
import api from '../config/axios';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .get('/auth/me')
      .then((r) => setUser(r.data.user))
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
  }, []);

  // emails are lowercased here, before they ever leave the browser
  const login = useCallback(async (email, password) => {
    const r = await api.post('/auth/login', { email: email.trim().toLowerCase(), password });
    setUser(r.data.user);
  }, []);

  const signup = useCallback(async (email, password, username) => {
    const r = await api.post('/auth/signup', {
      email: email.trim().toLowerCase(),
      password,
      username: username.trim(),
    });
    setUser(r.data.user);
  }, []);

  const logout = useCallback(async () => {
    await api.post('/auth/logout').catch(() => {});
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider value={{ user, loading, login, signup, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => useContext(AuthContext);
