import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import api, { errMsg } from '../config/axios';
import ErrorBox from '../components/ErrorBox';
import { auditSentence, fullTime, taskRef, timeAgo } from '../utils/format';

export default function Activity() {
  const { id } = useParams();
  const [entries, setEntries] = useState(null);
  const [error, setError] = useState('');

  useEffect(() => {
    api.get(`/projects/${id}/audit`).then((r) => setEntries(r.data.entries)).catch((e) => setError(errMsg(e)));
  }, [id]);

  return (
    <>
      <h2 className="mb">Activity</h2>
      <ErrorBox error={error} />
      {entries === null ? <p className="muted">loading…</p> : entries.length === 0 ? (
        <div className="empty">Nothing has happened yet.</div>
      ) : (
        <ul className="audit">
          {entries.map((e) => (
            <li key={e.id}>
              {auditSentence(e)}
              {e.task_id && (
                <> · <Link to={`/projects/${id}/tasks/${e.task_id}`} className="mono">{taskRef(e.task_number)}</Link></>
              )}
              <span className="muted" title={fullTime(e.created_at)}> · {timeAgo(e.created_at)}</span>
            </li>
          ))}
        </ul>
      )}
    </>
  );
}
