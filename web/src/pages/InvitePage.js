import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import api, { errMsg } from '../config/axios';
import { useAuth } from '../context/AuthContext';
import ErrorBox from '../components/ErrorBox';

const ROLE_TEXT = { viewer: 'read-only access', editor: 'read & write access' };

// Landing page for the emailed invite link: /invite?token=...
export default function InvitePage() {
  const [params] = useSearchParams();
  const token = params.get('token') || '';
  const { user, loading, login, signup, logout } = useAuth();
  const nav = useNavigate();
  const [invite, setInvite] = useState(null);
  const [error, setError] = useState('');
  const [password, setPassword] = useState('');
  const [username, setUsername] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!token) { setError('This invite link is missing its token.'); return; }
    api.get('/invites/preview', { params: { token } })
      .then((r) => setInvite(r.data))
      .catch((e) => setError(errMsg(e)));
  }, [token]);

  const accept = async () => {
    setBusy(true);
    setError('');
    try {
      const r = await api.post('/invites/accept', { token });
      nav(`/projects/${r.data.project_id}`, { replace: true });
    } catch (e) {
      setError(errMsg(e));
      setBusy(false);
    }
  };

  // log in or sign up with the invited email, then accept straight away
  const authAndAccept = async (e) => {
    e.preventDefault();
    setBusy(true);
    setError('');
    try {
      if (invite.has_account) await login(invite.email, password);
      else await signup(invite.email, password, username);
      const r = await api.post('/invites/accept', { token });
      nav(`/projects/${r.data.project_id}`, { replace: true });
    } catch (err) {
      setError(errMsg(err));
      setBusy(false);
    }
  };

  if (loading) return <div className="page narrow"><p className="muted">loading…</p></div>;
  if (!invite) return <div className="page narrow"><div className="card stack"><h1>Invite</h1><ErrorBox error={error} />{!error && <p className="muted">checking your invite…</p>}</div></div>;

  const wrongAccount = user && user.email.toLowerCase() !== invite.email;

  return (
    <div className="page narrow">
      <div className="card stack">
        <h1>Join {invite.project_name}</h1>
        <p><b>{invite.invited_by}</b> invited <b>{invite.email}</b> with {ROLE_TEXT[invite.role]}.</p>
        <ErrorBox error={error} />

        {user && !wrongAccount && (
          <button className="btn primary" onClick={accept} disabled={busy}>{busy ? 'Joining…' : 'Accept invite'}</button>
        )}

        {wrongAccount && (
          <>
            <p className="small">You're logged in as {user.email}, but this invite is for {invite.email}.</p>
            <button className="btn" onClick={logout}>Log out and continue</button>
          </>
        )}

        {!user && (
          <form className="stack" onSubmit={authAndAccept}>
            <label className="field">Email
              <input type="email" value={invite.email} disabled />
            </label>
            {!invite.has_account && (
              <label className="field">Username <span className="muted">optional, must be unique</span>
                <input value={username} onChange={(e) => setUsername(e.target.value)} pattern="[A-Za-z0-9_.\-]{3,32}" title="3-32 characters: letters, numbers, . _ -" />
              </label>
            )}
            <label className="field">{invite.has_account ? 'Password' : 'Choose a password'} {!invite.has_account && <span className="muted">at least 8 characters</span>}
              <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} minLength={invite.has_account ? undefined : 8} maxLength={72} required autoFocus />
            </label>
            <button className="btn primary" disabled={busy}>
              {busy ? 'Working…' : invite.has_account ? 'Log in & join' : 'Create account & join'}
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
