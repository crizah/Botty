import { useCallback, useEffect, useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import api, { errMsg } from '../config/axios';
import ErrorBox from './ErrorBox';
import { auditSentence, fullTime, memberName, taskRef, timeAgo } from '../utils/format';

// Side panel for one task: its reports (each with a status toggle + notes), task notes and activity.
export default function TaskDrawer({ taskId, onClose, onChanged }) {
  const { canWrite } = useOutletContext();
  const [data, setData] = useState(null);
  const [error, setError] = useState('');
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');

  const load = useCallback(async () => {
    try {
      const r = await api.get(`/tasks/${taskId}`);
      setData(r.data);
      setError('');
    } catch (e) {
      setError(errMsg(e));
    }
  }, [taskId]);

  useEffect(() => { setData(null); setEditing(false); load(); }, [load]);

  // close on Escape
  useEffect(() => {
    const onKey = (e) => e.key === 'Escape' && onClose();
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const refresh = async () => { await load(); onChanged(); };
  const run = async (fn) => {
    try { await fn(); await refresh(); } catch (e) { setError(errMsg(e)); }
  };

  if (!data) {
    return (
      <aside className="drawer">
        <div className="row between"><span className="muted">{error ? '' : 'loading…'}</span><button className="btn sm" onClick={onClose}>Close</button></div>
        <ErrorBox error={error} />
      </aside>
    );
  }

  const { task, reports, comments, audit } = data;
  const taskComments = comments.filter((c) => !c.report_id);

  const startEdit = () => { setTitle(task.title); setDescription(task.description); setEditing(true); };
  const saveEdit = (e) => {
    e.preventDefault();
    run(async () => {
      await api.patch(`/tasks/${task.id}`, { title, description });
      setEditing(false);
    });
  };

  return (
    <aside className="drawer" aria-label="Task details">
      <div className="row between">
        <div className="row">
          <span className="mono bold">{taskRef(task.number)}</span>
          <span className={`tag ${task.status}`}>{task.status}</span>
        </div>
        <button className="btn sm" onClick={onClose}>Close</button>
      </div>
      <ErrorBox error={error} />

      {editing ? (
        <form className="stack mt" onSubmit={saveEdit}>
          <input value={title} onChange={(e) => setTitle(e.target.value)} maxLength={200} required autoFocus />
          <textarea value={description} onChange={(e) => setDescription(e.target.value)} />
          <div className="row"><button className="btn primary">Save</button><button type="button" className="btn" onClick={() => setEditing(false)}>Cancel</button></div>
        </form>
      ) : (
        <div className="stack sm mt">
          <h1>{task.title}</h1>
          {task.description ? <p style={{ whiteSpace: 'pre-wrap' }}>{task.description}</p> : <p className="muted">No description.</p>}
          {canWrite && <div><button className="btn sm" onClick={startEdit}>Edit</button></div>}
        </div>
      )}

      {reports.length === 0 && (
        <div className="card dashed mt">
          <p className="small">No reports yet, so this task's status is set by hand.</p>
          <div className="mt">
            <button className="btn" disabled={!canWrite}
              onClick={() => run(() => api.patch(`/tasks/${task.id}/status`, { status: task.status === 'done' ? 'todo' : 'done' }))}>
              {task.status === 'done' ? 'Reopen task' : 'Mark task done'}
            </button>
          </div>
        </div>
      )}

      <hr />
      <div className="row between">
        <h2>Reports ({reports.filter((r) => r.status === 'done').length}/{reports.length} done)</h2>
      </div>
      <p className="muted">The task is done only when every report is done. Reopening any report reopens the task.</p>
      <div className="stack mt">
        {reports.length === 0 && <div className="empty">No reports.</div>}
        {reports.map((r) => (
          <ReportCard key={r.id} report={r} comments={comments.filter((c) => c.report_id === r.id)} canWrite={canWrite} run={run} />
        ))}
        {canWrite && <AddReport task={task} run={run} />}
      </div>

      <hr />
      <h2>Notes</h2>
      <Comments comments={taskComments} canWrite={canWrite} onAdd={(body) => run(() => api.post(`/tasks/${task.id}/comments`, { body }))} />

      <hr />
      <h2>Activity</h2>
      <ul className="audit mt">
        {audit.map((e) => (
          <li key={e.id}>
            {auditSentence(e)}
            <span className="muted" title={fullTime(e.created_at)}> · {timeAgo(e.created_at)}</span>
          </li>
        ))}
      </ul>
    </aside>
  );
}

function ReportCard({ report: r, comments, canWrite, run }) {
  const [showNotes, setShowNotes] = useState(false);
  const toggle = () => run(() => api.patch(`/reports/${r.id}/status`, { status: r.status === 'done' ? 'todo' : 'done' }));

  return (
    <div className="card">
      <div className="row between">
        <div className="row wrap">
          <span className={`tag ${r.status}`}>{r.status}</span>
          <span className="bold">{r.reporter_name || 'Unknown reporter'}</span>
          {r.group_name && <span className="muted">in {r.group_name}</span>}
        </div>
        <button className="btn sm" disabled={!canWrite} onClick={toggle}>{r.status === 'done' ? 'Reopen' : 'Mark done'}</button>
      </div>

      {r.message_text && <p className="quote mt">{r.message_text}</p>}
      <p className="muted mt">
        {r.message_id && <>message <span className="mono">{r.message_id}</span> · </>}
        {r.assigned_by_name ? <>assigned by {r.assigned_by_name}</> : 'assigned by —'}
        {r.status === 'done' && (
          <> · closed by {r.closed_by_user_name || r.closed_by_member_name || '—'}{r.closed_via && ` via ${r.closed_via}`}{r.closed_at && `, ${timeAgo(r.closed_at)}`}</>
        )}
      </p>

      <button className="btn link small mt" onClick={() => setShowNotes((s) => !s)}>
        {showNotes ? 'Hide notes' : `Notes (${comments.length})`}
      </button>
      {showNotes && (
        <div className="mt">
          <Comments comments={comments} canWrite={canWrite} onAdd={(body) => run(() => api.post(`/reports/${r.id}/comments`, { body }))} />
        </div>
      )}
    </div>
  );
}

function Comments({ comments, canWrite, onAdd }) {
  const [body, setBody] = useState('');
  const submit = async (e) => {
    e.preventDefault();
    if (!body.trim()) return;
    await onAdd(body);
    setBody('');
  };
  return (
    <div className="stack sm mt">
      {comments.length === 0 && <p className="muted">No notes yet.</p>}
      {comments.map((c) => (
        <div className="comment" key={c.id}>
          <p className="small"><span className="bold">{c.author_name || 'Someone'}</span> <span className="muted" title={fullTime(c.created_at)}>{timeAgo(c.created_at)}</span></p>
          <p style={{ whiteSpace: 'pre-wrap' }}>{c.body}</p>
        </div>
      ))}
      {canWrite && (
        <form className="row" onSubmit={submit}>
          <input placeholder="Add a note…" value={body} onChange={(e) => setBody(e.target.value)} maxLength={5000} />
          <button className="btn" disabled={!body.trim()}>Add</button>
        </form>
      )}
    </div>
  );
}

// Manual report entry.
// TODO: change - the ingest pipeline will create reports from tagged messages; this form is
// for testing the dashboard (and for hand-adding a request that never hit WhatsApp).
function AddReport({ task, run }) {
  const [open, setOpen] = useState(false);
  const [groups, setGroups] = useState([]);
  const [members, setMembers] = useState([]);
  const [groupId, setGroupId] = useState('');
  const [reporter, setReporter] = useState('');
  const [assignedBy, setAssignedBy] = useState('');
  const [text, setText] = useState('');
  const [messageId, setMessageId] = useState('');

  useEffect(() => {
    if (open) api.get(`/projects/${task.project_id}/groups`).then((r) => setGroups(r.data.groups)).catch(() => {});
  }, [open, task.project_id]);

  useEffect(() => {
    setReporter(''); setAssignedBy('');
    if (!groupId) { setMembers([]); return; }
    api.get(`/projects/${task.project_id}/groups/${groupId}/members`).then((r) => setMembers(r.data.members)).catch(() => {});
  }, [groupId, task.project_id]);

  const submit = (e) => {
    e.preventDefault();
    run(async () => {
      await api.post(`/tasks/${task.id}/reports`, {
        group_id: groupId, reporter_member_id: reporter, assigned_by_member_id: assignedBy,
        message_text: text, message_id: messageId,
      });
      setText(''); setMessageId(''); setOpen(false);
    });
  };

  if (!open) return <button className="btn" onClick={() => setOpen(true)}>Add report manually</button>;

  const memberOptions = members.map((m) => <option key={m.id} value={m.id}>{memberName(m)}</option>);
  return (
    <form className="card dashed stack" onSubmit={submit}>
      <label className="field">Group
        <select value={groupId} onChange={(e) => setGroupId(e.target.value)}>
          <option value="">No group</option>
          {groups.map((g) => <option key={g.id} value={g.id}>{g.name}</option>)}
        </select>
      </label>
      <div className="row">
        <label className="field" style={{ flex: 1 }}>Reported by
          <select value={reporter} onChange={(e) => setReporter(e.target.value)} disabled={!groupId}><option value="">—</option>{memberOptions}</select>
        </label>
        <label className="field" style={{ flex: 1 }}>Assigned by (tagged botty)
          <select value={assignedBy} onChange={(e) => setAssignedBy(e.target.value)} disabled={!groupId}><option value="">—</option>{memberOptions}</select>
        </label>
      </div>
      <label className="field">Message
        <textarea value={text} onChange={(e) => setText(e.target.value)} />
      </label>
      <label className="field">WhatsApp message id <span className="muted">optional</span>
        <input value={messageId} onChange={(e) => setMessageId(e.target.value)} />
      </label>
      <div className="row"><button className="btn primary">Add report</button><button type="button" className="btn" onClick={() => setOpen(false)}>Cancel</button></div>
    </form>
  );
}
