import { useCallback, useEffect, useState } from 'react';
import { NavLink, Outlet, useParams } from 'react-router-dom';
import api, { errMsg } from '../config/axios';
import ErrorBox from '../components/ErrorBox';

const MODES = [
  { value: 'bot_only', label: 'Bot only', hint: 'Only messages that tag @botty are turned into tasks.' },
  { value: 'all_texts', label: 'All texts', hint: 'Botty reads every message in the attached groups.' },
];

// Header + tabs for one project. Child pages get { project, role, canWrite, isOwner, reload } via outlet context.
export default function ProjectLayout() {
  const { id } = useParams();
  const [data, setData] = useState(null);
  const [error, setError] = useState('');

  const reload = useCallback(async () => {
    try {
      const r = await api.get(`/projects/${id}`);
      setData(r.data);
      setError('');
    } catch (e) {
      setError(errMsg(e));
    }
  }, [id]);

  useEffect(() => { reload(); }, [reload]);

  if (error && !data) return <div className="page"><ErrorBox error={error} /></div>;
  if (!data) return <div className="page"><p className="muted">loading…</p></div>;

  const { project, role } = data;
  const isOwner = role === 'owner';
  const canWrite = role === 'owner' || role === 'editor';

  const setMode = async (mode) => {
    try {
      await api.patch(`/projects/${id}`, { listen_mode: mode });
      reload();
    } catch (e) {
      setError(errMsg(e));
    }
  };

  const tab = ({ isActive }) => (isActive ? 'active' : '');

  return (
    <div className="page">
      <div className="row between wrap">
        <div>
          <h1>{project.name}</h1>
          {project.description && <p className="small">{project.description}</p>}
        </div>
        <div className="stack sm" style={{ alignItems: 'flex-end' }}>
          <div className="row">
            <span className="small bold">Listening</span>
            <div className="seg">
              {MODES.map((m) => (
                <button key={m.value} className={project.listen_mode === m.value ? 'active' : ''}
                  disabled={!isOwner} title={m.hint} onClick={() => setMode(m.value)}>
                  {m.label}
                </button>
              ))}
            </div>
          </div>
          <span className="muted">{MODES.find((m) => m.value === project.listen_mode)?.hint}</span>
        </div>
      </div>

      <div className="tabs">
        <NavLink to={`/projects/${id}`} end className={tab}>Board</NavLink>
        <NavLink to={`/projects/${id}/groups`} className={tab}>Groups</NavLink>
        <NavLink to={`/projects/${id}/activity`} className={tab}>Activity</NavLink>
        <span className="spacer" />
        <span className="tag" title="your role in this project">{role}</span>
      </div>

      <ErrorBox error={error} />
      <Outlet context={{ project, role, canWrite, isOwner, reload }} />
    </div>
  );
}
