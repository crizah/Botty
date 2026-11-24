import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import api, { errMsg } from '../config/axios';
import ErrorBox from '../components/ErrorBox';

// Shows the bot's number and the groups it has been added to.
// TODO: change - polling every 5s is a stand-in; the ingest worker will push group joins.
export default function Groups() {
  const [bot, setBot] = useState(null);
  const [groups, setGroups] = useState(null);
  const [error, setError] = useState('');
  const [devName, setDevName] = useState('');

  const load = useCallback(async () => {
    try {
      const r = await api.get('/groups');
      setGroups(r.data.groups);
    } catch (e) {
      setError(errMsg(e));
    }
  }, []);

  useEffect(() => {
    api.get('/bot').then((r) => setBot(r.data)).catch((e) => setError(errMsg(e)));
    load();
    const t = setInterval(load, 5000);
    return () => clearInterval(t);
  }, [load]);

  const simulate = async (e) => {
    e.preventDefault();
    try {
      await api.post('/dev/groups', { name: devName.trim() });
      setDevName('');
      load();
    } catch (err) {
      setError(errMsg(err));
    }
  };

  return (
    <div className="page">
      <h1 className="mb">Groups</h1>
      <ErrorBox error={error} />

      <div className="card mb">
        <h2>1. Add the bot to your WhatsApp group</h2>
        <p className="mt small">Open the group, tap Add member, and add this number:</p>
        <p className="big-number mono mt">{bot ? bot.number : '…'}</p>
        <p className="muted mt">Groups the bot joins appear below automatically. Nothing is read until you attach the group to a project.</p>
      </div>

      {bot?.dev_mode && (
        <form className="card dashed row mb" onSubmit={simulate}>
          {/* TODO: change - dev only; remove when real group discovery is wired up */}
          <span className="small bold">DEV</span>
          <input placeholder="Group name (optional)" value={devName} onChange={(e) => setDevName(e.target.value)} />
          <button className="btn">Simulate bot added to a group</button>
        </form>
      )}

      <h2 className="mb">2. Your groups</h2>
      {groups === null ? (
        <p className="muted">loading…</p>
      ) : groups.length === 0 ? (
        <div className="empty">No groups yet. Add the number above to a group and it will show up here.</div>
      ) : (
        <div className="grid">
          {groups.map((g) => (
            <div className="card" key={g.id}>
              <h3>{g.name}</h3>
              <p className="muted">{g.member_count} members</p>
              <p className="mt small">
                {g.project_id ? (
                  <>In a project · <Link to={`/projects/${g.project_id}/groups`}>open</Link></>
                ) : (
                  <span className="tag dashed">not in a project</span>
                )}
              </p>
            </div>
          ))}
        </div>
      )}
      {groups?.length > 0 && <p className="mt"><Link to="/">Go to projects →</Link></p>}
    </div>
  );
}
