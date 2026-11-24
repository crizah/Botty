import { useCallback, useEffect, useState } from 'react';
import { useOutletContext, useParams } from 'react-router-dom';
import api, { errMsg } from '../config/axios';
import ErrorBox from '../components/ErrorBox';
import { fullTime } from '../utils/format';

const ROLE_LABEL = { owner: 'Owner', editor: 'Write', viewer: 'Read' };

// Who is in the project. Everyone can see members; only the owner can invite, change access or remove.
export default function ProjectUsers() {
  const { id } = useParams();
  const { isOwner } = useOutletContext();
  const [data, setData] = useState(null);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [email, setEmail] = useState('');
  const [role, setRole] = useState('viewer');
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const r = await api.get(`/projects/${id}/users`);
      setData(r.data);
    } catch (e) {
      setError(errMsg(e));
    }
  }, [id]);

  useEffect(() => { load(); }, [load]);

  // runs an owner action, then reloads the lists
  const act = async (fn, done) => {
    setError('');
    setNotice('');
    try {
      const r = await fn();
      if (done) setNotice(done);
      await load();
      return r;
    } catch (e) {
      setError(errMsg(e));
    }
  };

  const invite = async (e) => {
    e.preventDefault();
    setBusy(true);
    const r = await act(() => api.post(`/projects/${id}/invites`, { email: email.trim().toLowerCase(), role }), `Invite sent to ${email.trim().toLowerCase()}`);
    if (r) setEmail('');
    setBusy(false);
  };

  if (!data) return error ? <ErrorBox error={error} /> : <p className="muted">loading…</p>;

  return (
    <div className="stack">
      <ErrorBox error={error} />
      {notice && <div className="card small">{notice}</div>}
      {isOwner && (
        <form className="card stack" onSubmit={invite}>
          <h2>Invite someone</h2>
          <p className="muted">They get an email with a link that works once and expires in 48 hours.</p>
          <div className="row wrap">
            <input type="email" placeholder="email@example.com" value={email} onChange={(e) => setEmail(e.target.value)} required style={{ flex: 1, minWidth: 220 }} />
            <select value={role} onChange={(e) => setRole(e.target.value)} aria-label="access">
              <option value="viewer">Read only</option>
              <option value="editor">Read &amp; write</option>
            </select>
            <button className="btn primary" disabled={busy}>{busy ? 'Sending…' : 'Send invite'}</button>
          </div>
        </form>
      )}

      <div className="card stack">
        <h2>Members <span className="tag">{data.members.length}</span></h2>
        {data.members.map((m) => (
          <div className="row between wrap" key={m.user_id}>
            <div>
              <div className="bold">{m.username || m.email}</div>
              {m.username && <div className="muted">{m.email}</div>}
            </div>
            <div className="row">
              {isOwner && m.role !== 'owner' ? (
                <>
                  <select value={m.role} aria-label={`access for ${m.email}`}
                    onChange={(e) => act(() => api.patch(`/projects/${id}/users/${m.user_id}`, { role: e.target.value }))}>
                    <option value="viewer">Read only</option>
                    <option value="editor">Read &amp; write</option>
                  </select>
                  <button className="btn" onClick={() => window.confirm(`Remove ${m.email} from this project?`) && act(() => api.delete(`/projects/${id}/users/${m.user_id}`))}>Remove</button>
                </>
              ) : (
                <span className="tag">{ROLE_LABEL[m.role]}</span>
              )}
            </div>
          </div>
        ))}
      </div>

      {isOwner && data.invites.length > 0 && (
        <div className="card stack">
          <h2>Pending invites <span className="tag">{data.invites.length}</span></h2>
          {data.invites.map((i) => {
            const expired = new Date(i.expires_at) < new Date();
            return (
              <div className="row between wrap" key={i.id}>
                <div>
                  <div className="bold">{i.email}</div>
                  <div className="muted">{ROLE_LABEL[i.role]} · {expired ? 'expired' : `expires ${fullTime(i.expires_at)}`}</div>
                </div>
                <div className="row">
                  <button className="btn" onClick={() => act(() => api.post(`/projects/${id}/invites/${i.id}/resend`), `Invite re-sent to ${i.email}`)}>Resend</button>
                  <button className="btn" onClick={() => act(() => api.delete(`/projects/${id}/invites/${i.id}`))}>Revoke</button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
