import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useOutletContext, useParams } from 'react-router-dom';
import api, { errMsg } from '../config/axios';
import ErrorBox from '../components/ErrorBox';
import TaskDrawer from '../components/TaskDrawer';
import { taskRef } from '../utils/format';

const COLUMNS = [
  { status: 'todo', title: 'Todo' },
  { status: 'done', title: 'Done' },
];

export default function Board() {
  const { id, taskId } = useParams();
  const { canWrite } = useOutletContext();
  const nav = useNavigate();
  const [tasks, setTasks] = useState(null);
  const [error, setError] = useState('');
  const [adding, setAdding] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');

  const load = useCallback(async () => {
    try {
      const r = await api.get(`/projects/${id}/tasks`);
      setTasks(r.data.tasks);
    } catch (e) {
      setError(errMsg(e));
    }
  }, [id]);

  useEffect(() => { load(); }, [load]);

  const refresh = async () => {
    setRefreshing(true);
    setError('');
    await load();
    setRefreshing(false);
  };

  const create = async (e) => {
    e.preventDefault();
    setError('');
    try {
      const r = await api.post(`/projects/${id}/tasks`, { title, description });
      setTitle('');
      setDescription('');
      setAdding(false);
      await load();
      nav(`/projects/${id}/tasks/${r.data.task.id}`);
    } catch (err) {
      setError(errMsg(err));
    }
  };

  return (
    <>
      <div className="row between mb">
        <h2>Tasks</h2>
        <div className="row">
          <button className="btn" onClick={refresh} disabled={refreshing}>{refreshing ? 'Refreshing…' : 'Refresh'}</button>
          {canWrite && <button className="btn" onClick={() => setAdding((a) => !a)}>{adding ? 'Cancel' : 'New task'}</button>}
        </div>
      </div>
      <ErrorBox error={error} />

      {adding && (
        <form className="card stack mb" onSubmit={create}>
          <label className="field">Task name
            <input value={title} onChange={(e) => setTitle(e.target.value)} maxLength={200} required autoFocus />
          </label>
          <label className="field">Description <span className="muted">optional</span>
            <textarea value={description} onChange={(e) => setDescription(e.target.value)} />
          </label>
          <div><button className="btn primary">Create task</button></div>
        </form>
      )}

      {tasks === null ? (
        <p className="muted">loading…</p>
      ) : (
        <div className="board">
          {COLUMNS.map((col) => {
            const items = tasks.filter((t) => t.status === col.status);
            return (
              <section key={col.status}>
                <div className="col-head">
                  <h2>{col.title}</h2>
                  <span className="tag">{items.length}</span>
                </div>
                <div className="stack">
                  {items.length === 0 && <div className="empty">Nothing here</div>}
                  {items.map((t) => (
                    <Link key={t.id} to={`/projects/${id}/tasks/${t.id}`} className="card click">
                      <div className="row between">
                        <span className="small bold mono">{taskRef(t.number)}</span>
                        <span className={`tag ${t.status}`}>{t.status}</span>
                      </div>
                      <h3 className="mt">{t.title}</h3>
                      {t.description && <p className="small task-desc">{t.description}</p>}
                      {t.report_count > 0 ? (
                        <>
                          <div className="progress" title={`${t.done_count} of ${t.report_count} reports done`}>
                            {Array.from({ length: t.report_count }, (_, i) => (
                              <span key={i} className={i < t.done_count ? 'on' : ''} />
                            ))}
                          </div>
                          <p className="small">{t.done_count}/{t.report_count} reports done{t.comment_count > 0 && ` · ${t.comment_count} notes`}</p>
                        </>
                      ) : (
                        <p className="muted">no reports yet{t.comment_count > 0 && ` · ${t.comment_count} notes`}</p>
                      )}
                    </Link>
                  ))}
                </div>
              </section>
            );
          })}
        </div>
      )}

      {taskId && <TaskDrawer taskId={taskId} onClose={() => nav(`/projects/${id}`)} onChanged={load} />}
    </>
  );
}
