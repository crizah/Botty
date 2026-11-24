import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import api, { errMsg } from '../config/axios';
import { useData } from '../context/DataContext';
import ErrorBox from '../components/ErrorBox';

// Shows the bot's number and the groups it has been added to.
export default function Groups() {
  const { groups, refresh } = useData(); // refreshed every 30s by the data provider
  const [bot, setBot] = useState(null);
  const [error, setError] = useState('');
  const [devName, setDevName] = useState('');

  useEffect(() => {
    api.get('/bot').then((r) => setBot(r.data)).catch((e) => setError(errMsg(e)));
    refresh();
  }, [refresh]);

  const [claim, setClaim] = useState(null);
  const [copied, setCopied] = useState(false);

  const makeCode = async () => {
    setError('');
    try {
      const r = await api.post('/groups/claims');
      setClaim(r.data);
      setCopied(false);
    } catch (err) {
      setError(errMsg(err));
    }
  };

  const copy = () => {
    navigator.clipboard?.writeText(claim.command);
    setCopied(true);
  };

  // TODO: change - dev only; the real bot redeems the code when it sees "@botty link <code>" in a group
  const simulate = async (e) => {
    e.preventDefault();
    try {
      await api.post('/dev/groups/link', { code: claim.code, name: devName.trim() });
      setDevName('');
      setClaim(null);
      refresh();
    } catch (err) {
      setError(errMsg(err));
    }
  };

  return (
    <div className="page">
      <h1 className="mb">Groups</h1>
      <ErrorBox error={error} />

      <div className="card mb">
        <h2>1. Add the bot and link the group</h2>
        <p className="mt small">Add this number to your WhatsApp group:</p>
        <p className="big-number mono mt">{bot ? bot.number : '…'}</p>
        <p className="mt small">Then generate a code and send it in the group. The bot ignores groups until they're linked.</p>
        {claim ? (
          <div className="stack mt">
            <p className="big-number mono">{claim.command}</p>
            <div className="row">
              <button className="btn" onClick={copy}>{copied ? 'Copied' : 'Copy'}</button>
              <button className="btn" onClick={makeCode}>New code</button>
            </div>
            <p className="muted small">Single use, expires in 30 minutes.</p>
          </div>
        ) : (
          <div className="stack mt"><button className="btn primary" onClick={makeCode}>Generate link code</button></div>
        )}
      </div>

      {bot?.dev_mode && claim && (
        <form className="card dashed row mb" onSubmit={simulate}>
          {/* TODO: change - dev only; remove when the bot handles real messages */}
          <span className="small bold">DEV</span>
          <input placeholder="Group name (optional)" value={devName} onChange={(e) => setDevName(e.target.value)} />
          <button className="btn">Simulate sending the command in a group</button>
        </form>
      )}

      <h2 className="mb">2. Your groups</h2>
      {groups === null ? (
        <p className="muted">loading…</p>
      ) : groups.length === 0 ? (
        <div className="empty">No groups yet. Add the number to a group and send the link command.</div>
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
