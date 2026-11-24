import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import api, { errMsg } from '../config/axios';
import ErrorBox from '../components/ErrorBox';

export default function Projects() {
  const nav = useNavigate();
  const [projects, setProjects] = useState(null);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    api.get('/projects').then((r) => setProjects(r.data.projects)).catch((e) => setError(errMsg(e)));
  }, []);

  const create = async (e) => {
    e.preventDefault();
    setError('');
    try {
      const r = await api.post('/projects', { name, description });
      nav(`/projects/${r.data.project.id}/groups`); // first thing to do: attach a group
    } catch (err) {
      setError(errMsg(err));
    }
  };

  return (
    <div className="page">
      <div className="row between mb">
        <h1>Projects</h1>
        <button className="btn primary" onClick={() => setCreating((c) => !c)}>{creating ? 'Cancel' : 'New project'}</button>
      </div>
      <ErrorBox error={error} />

      {creating && (
        <form className="card stack mb" onSubmit={create}>
          <label className="field">Name
            <input value={name} onChange={(e) => setName(e.target.value)} maxLength={100} required autoFocus />
          </label>
          <label className="field">Description <span className="muted">optional</span>
            <textarea value={description} onChange={(e) => setDescription(e.target.value)} />
          </label>
          <div><button className="btn primary">Create project</button></div>
        </form>
      )}

      {projects === null ? (
        <p className="muted">loading…</p>
      ) : projects.length === 0 ? (
        <div className="empty">No projects yet. Create one, then attach the groups you want it to track.</div>
      ) : (
        <div className="grid">
          {projects.map((p) => (
            <Link className="card click" key={p.id} to={`/projects/${p.id}`}>
              <div className="row between">
                <h2>{p.name}</h2>
                <span className="tag">{p.role}</span>
              </div>
              {p.description && <p className="small mt">{p.description}</p>}
              <p className="small mt">
                {p.task_count} tasks · {p.done_count} done · {p.group_count} groups
              </p>
              <p className="muted">listening: {p.listen_mode === 'bot_only' ? 'bot only' : 'all texts'}</p>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
