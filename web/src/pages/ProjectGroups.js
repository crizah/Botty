import { useCallback, useEffect, useState } from 'react';
import { useOutletContext, useParams } from 'react-router-dom';
import { Link } from 'react-router-dom';
import api, { errMsg } from '../config/axios';
import ErrorBox from '../components/ErrorBox';
import { memberName } from '../utils/format';

export default function ProjectGroups() {
  const { id } = useParams();
  const { isOwner } = useOutletContext();
  const [attached, setAttached] = useState(null);
  const [available, setAvailable] = useState([]);
  const [pick, setPick] = useState('');
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    try {
      const [a, all] = await Promise.all([api.get(`/projects/${id}/groups`), api.get('/groups')]);
      setAttached(a.data.groups);
      // the dropdown: groups the bot is in that aren't part of any project yet
      setAvailable(all.data.groups.filter((g) => !g.project_id));
    } catch (e) {
      setError(errMsg(e));
    }
  }, [id]);

  useEffect(() => { load(); }, [load]);

  const attach = async () => {
    if (!pick) return;
    try {
      await api.post(`/projects/${id}/groups`, { group_id: pick });
      setPick('');
      load();
    } catch (e) {
      setError(errMsg(e));
    }
  };

  const detach = async (gid) => {
    if (!window.confirm('Remove this group from the project? Its tag allowlist is removed too.')) return;
    try {
      await api.delete(`/projects/${id}/groups/${gid}`);
      load();
    } catch (e) {
      setError(errMsg(e));
    }
  };

  return (
    <>
      <h2 className="mb">Groups in this project</h2>
      <ErrorBox error={error} />

      {isOwner && (
        <div className="card row mb">
          <select value={pick} onChange={(e) => setPick(e.target.value)} aria-label="Group to add">
            <option value="">Select a group the bot is in…</option>
            {available.map((g) => <option key={g.id} value={g.id}>{g.name} ({g.member_count} members)</option>)}
          </select>
          <button className="btn primary" onClick={attach} disabled={!pick}>Add</button>
        </div>
      )}
      {isOwner && available.length === 0 && (
        <p className="muted mb">No free groups. Add the bot to a group first — see <Link to="/groups">Groups</Link>.</p>
      )}

      {attached === null ? <p className="muted">loading…</p> : attached.length === 0 ? (
        <div className="empty">No groups attached. Attach one so messages from it can become tasks in this project.</div>
      ) : (
        <div className="stack">
          {attached.map((g) => (
            <GroupCard key={g.id} projectId={id} group={g} isOwner={isOwner} onDetach={() => detach(g.id)} onChanged={load} />
          ))}
        </div>
      )}
    </>
  );
}

// One attached group + its "who may tag @botty" allowlist.
function GroupCard({ projectId, group, isOwner, onDetach, onChanged }) {
  const base = `/projects/${projectId}/groups/${group.id}`;
  const [open, setOpen] = useState(false);
  const [members, setMembers] = useState([]);
  const [allowed, setAllowed] = useState([]);
  const [pick, setPick] = useState('');
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    try {
      const [m, a] = await Promise.all([api.get(`${base}/members`), api.get(`${base}/allowlist`)]);
      setMembers(m.data.members);
      setAllowed(a.data.members);
    } catch (e) {
      setError(errMsg(e));
    }
  }, [base]);

  useEffect(() => { if (open) load(); }, [open, load]);

  const allowedIds = new Set(allowed.map((m) => m.id));
  const candidates = members.filter((m) => !allowedIds.has(m.id));

  const add = async () => {
    if (!pick) return;
    try {
      await api.post(`${base}/allowlist`, { member_id: pick });
      setPick('');
      await load();
      onChanged();
    } catch (e) {
      setError(errMsg(e));
    }
  };

  const remove = async (mid) => {
    try {
      await api.delete(`${base}/allowlist/${mid}`);
      await load();
      onChanged();
    } catch (e) {
      setError(errMsg(e));
    }
  };

  return (
    <div className="card">
      <div className="row between">
        <div>
          <h3>{group.name}</h3>
          <p className="muted">{group.member_count} members · {group.allowlist_count} allowed to tag @botty</p>
        </div>
        <div className="row">
          <button className="btn sm" onClick={() => setOpen((o) => !o)}>{open ? 'Hide allowlist' : 'Tag allowlist'}</button>
          {isOwner && <button className="btn sm" onClick={onDetach}>Remove</button>}
        </div>
      </div>

      {open && (
        <div className="stack mt">
          <hr className="dashed" style={{ margin: 0 }} />
          <ErrorBox error={error} />
          <p className="small">Only these people can make @botty act in this group.</p>
          {allowed.length === 0 ? (
            <p className="muted">Nobody yet.</p>
          ) : (
            <ul style={{ margin: 0, padding: 0, listStyle: 'none' }} className="stack sm">
              {allowed.map((m) => (
                <li key={m.id} className="row between">
                  <span>
                    {memberName(m)}
                    {/* TODO: change - phone numbers stay empty until the identity pipeline can resolve WhatsApp ids */}
                    {!m.phone && <span className="muted"> · number not resolved</span>}
                  </span>
                  {isOwner && <button className="btn sm" onClick={() => remove(m.id)}>Remove</button>}
                </li>
              ))}
            </ul>
          )}
          {isOwner && (
            <div className="row">
              <select value={pick} onChange={(e) => setPick(e.target.value)} aria-label="Member to allow">
                <option value="">Select a member…</option>
                {candidates.map((m) => <option key={m.id} value={m.id}>{memberName(m)}{m.phone && m.push_name ? ` (${m.phone})` : ''}</option>)}
              </select>
              <button className="btn" onClick={add} disabled={!pick}>Allow</button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
