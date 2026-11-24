import axios from 'axios';

// Backend URL, in order of priority:
//  1. build time:   REACT_APP_BACKEND_URL (web/.env, CRA bakes it in)
//  2. runtime:      window.RUNTIME_CONFIG from public/config.js (envsubst in Docker/nginx)
// "${...}" means config.js was served without substitution, so ignore it.
const runtimeUrl = window.RUNTIME_CONFIG?.REACT_APP_BACKEND_URL;
const BACKEND_URL =
  (runtimeUrl && !runtimeUrl.startsWith('${') ? runtimeUrl : null) ||
  process.env.REACT_APP_BACKEND_URL ||
  'http://localhost:8081'; // TODO: change - fallback for local dev only

const api = axios.create({
  baseURL: `${BACKEND_URL.replace(/\/$/, '')}/api/v1`,
  withCredentials: true, // the JWT lives in an httpOnly cookie
  headers: { 'Content-Type': 'application/json' },
});

api.interceptors.response.use(
  (res) => res,
  (error) => {
    const url = error.config?.url || '';
    // a 401 anywhere except the auth endpoints means the session is gone
    if (error.response?.status === 401 && !url.includes('/auth/')) {
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export const errMsg = (e) =>
  e?.response?.data?.error || (e?.request ? 'Cannot reach the server' : 'Something went wrong');

export default api;
