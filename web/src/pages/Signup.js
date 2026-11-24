import { useState } from 'react';
import { Link, Navigate, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { errMsg } from '../config/axios';
import ErrorBox from '../components/ErrorBox';

export default function Signup() {
  const { user, signup } = useAuth();
  const nav = useNavigate();
  const [email, setEmail] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  if (user) return <Navigate to="/groups" replace />;

  const submit = async (e) => {
    e.preventDefault();
    setBusy(true);
    setError('');
    try {
      await signup(email, password, username);
      nav('/groups'); // next step: add the bot to a group
    } catch (err) {
      setError(errMsg(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="page narrow">
      <form className="card stack" onSubmit={submit}>
        <h1>Create account</h1>
        <ErrorBox error={error} />
        <label className="field">Email
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoFocus />
        </label>
        <label className="field">Username <span className="muted">optional, must be unique</span>
          <input value={username} onChange={(e) => setUsername(e.target.value)} pattern="[A-Za-z0-9_.\-]{3,32}" title="3-32 characters: letters, numbers, . _ -" />
        </label>
        <label className="field">Password <span className="muted">at least 8 characters</span>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} minLength={8} maxLength={72} required />
        </label>
        <button className="btn primary" disabled={busy}>{busy ? 'Creating…' : 'Sign up'}</button>
        <p className="small">Already have an account? <Link to="/login">Log in</Link></p>
      </form>
    </div>
  );
}
