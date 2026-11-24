import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import api, { errMsg } from '../config/axios';
import { useData } from '../context/DataContext';
import ErrorBox from './ErrorBox';

export default function NewProjectModal() {
  const { setNewProjectOpen, refresh } = useData();
  const nav = useNavigate();
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const close = () => setNewProjectOpen(false);

  useEffect(() => {
    const onKey = (e) => e.key === 'Escape' && setNewProjectOpen(false);
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [setNewProjectOpen]);

  const submit = async (e) => {
    e.preventDefault();
    setBusy(true);
    setError('');
    try {
      const r = await api.post('/projects', { name, description });
      await refresh();
      close();
      nav(`/projects/${r.data.project.id}/groups`); // first step: attach a group
    } catch (err) {
      setError(errMsg(err));
      setBusy(false);
    }
  };

  return (
    <div className="modal-backdrop" onMouseDown={close}>
      <form className="card modal stack" onMouseDown={(e) => e.stopPropagation()} onSubmit={submit}>
        <div className="row between">
          <h2>New project</h2>
          <button type="button" className="btn ghost sm" onClick={close} aria-label="Close">✕</button>
        </div>
        <p className="muted">A project collects tasks from the WhatsApp groups you attach to it.</p>
        <ErrorBox error={error} />
        <label className="field">Name
          <input value={name} onChange={(e) => setName(e.target.value)} maxLength={100} required autoFocus />
        </label>
        <label className="field">Description <span className="muted">optional</span>
          <textarea value={description} onChange={(e) => setDescription(e.target.value)} />
        </label>
        <div className="row" style={{ justifyContent: 'flex-end' }}>
          <button type="button" className="btn" onClick={close}>Cancel</button>
          <button className="btn primary" disabled={busy}>{busy ? 'Creating…' : 'Create project'}</button>
        </div>
      </form>
    </div>
  );
}
