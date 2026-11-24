import { useEffect } from 'react';
import { Link } from 'react-router-dom';
import { useData } from '../context/DataContext';

export default function Dashboard() {
  const { projects, groups, refresh, setNewProjectOpen } = useData();

  useEffect(() => { refresh(); }, [refresh]); // task counts may have changed since the last visit

  const loading = projects === null || groups === null;
  const tasks = (projects || []).reduce((n, p) => n + p.task_count, 0);
  const done = (projects || []).reduce((n, p) => n + p.done_count, 0);
  const grouped = (groups || []).filter((g) => g.project_id).length;

  return (
    <div className="page">
      <div className="row between mb">
        <h1>Dashboard</h1>
        <button className="btn primary" onClick={() => setNewProjectOpen(true)}>+ New project</button>
      </div>

      <div className="stats mb">
        <Stat label="Projects" value={loading ? '–' : projects.length} />
        <Stat label="Groups added" value={loading ? '–' : groups.length} sub={loading ? '' : `${grouped} in a project`} />
        <Stat label="Open tasks" value={loading ? '–' : tasks - done} />
        <Stat label="Done tasks" value={loading ? '–' : done} />
      </div>

      <h2 className="mb">Projects</h2>
      {loading ? (
        <p className="muted">loading…</p>
      ) : projects.length === 0 ? (
        <div className="empty">
          <p>No projects yet.</p>
          <p className="mt"><button className="btn primary" onClick={() => setNewProjectOpen(true)}>Create your first project</button></p>
        </div>
      ) : (
        <div className="grid">
          {projects.map((p) => (
            <Link className="card click" key={p.id} to={`/projects/${p.id}`}>
              <div className="row between">
                <h3>{p.name}</h3>
                <span className="tag">{p.role}</span>
              </div>
              {p.description && <p className="small mt">{p.description}</p>}
              <div className="bar mt" title={`${p.done_count} of ${p.task_count} tasks done`}>
                <span style={{ width: p.task_count ? `${(p.done_count / p.task_count) * 100}%` : 0 }} />
              </div>
              <p className="small mt">{p.done_count}/{p.task_count} tasks done · {p.group_count} {p.group_count === 1 ? 'group' : 'groups'}</p>
            </Link>
          ))}
        </div>
      )}

      {!loading && groups.length === 0 && (
        <div className="card mt">
          <h3>Add the bot to a WhatsApp group</h3>
          <p className="small mt">Groups the bot joins show up in the sidebar. <Link to="/groups">See how →</Link></p>
        </div>
      )}
    </div>
  );
}

function Stat({ label, value, sub }) {
  return (
    <div className="card stat">
      <p className="small">{label}</p>
      <p className="stat-number">{value}</p>
      {sub && <p className="muted">{sub}</p>}
    </div>
  );
}
