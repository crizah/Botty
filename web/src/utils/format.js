export const taskRef = (n) => `T-${n}`;

export function timeAgo(iso) {
  const s = Math.max(0, (Date.now() - new Date(iso).getTime()) / 1000);
  if (s < 60) return 'just now';
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  if (s < 86400 * 7) return `${Math.floor(s / 86400)}d ago`;
  return new Date(iso).toLocaleDateString();
}

export const fullTime = (iso) => new Date(iso).toLocaleString();

export const memberName = (m) => m.push_name || m.phone || m.wa_id;

const MODE_LABEL = { bot_only: 'bot only', all_texts: 'all texts' };

// Kaneo-style sentence for an audit log entry: "<who> <did what>"
export function auditSentence(e) {
  const who = e.actor_kind === 'system' ? 'System' : e.actor_kind === 'bot' ? 'Botty' : e.actor_name || 'Someone';
  const q = (s) => (s ? `"${s}"` : '');
  switch (e.action) {
    case 'project.created':
      return `${who} created the project ${q(e.detail)}`;
    case 'project.renamed':
      return `${who} renamed the project from ${q(e.old_value)} to ${q(e.new_value)}`;
    case 'project.listen_mode_changed':
      return `${who} changed listening from ${MODE_LABEL[e.old_value] || e.old_value} to ${MODE_LABEL[e.new_value] || e.new_value}`;
    case 'project.group_added':
      return `${who} added the group ${q(e.detail)}`;
    case 'project.group_removed':
      return `${who} removed the group ${q(e.detail)}`;
    case 'allowlist.added':
      return `${who} allowed ${e.detail} to tag the bot`;
    case 'allowlist.removed':
      return `${who} removed ${e.detail} from the tag allowlist`;
    case 'task.created':
      return `${who} created the task ${q(e.detail)}`;
    case 'task.renamed':
      return `${who} renamed the task from ${q(e.old_value)} to ${q(e.new_value)}`;
    case 'task.description_changed':
      return `${who} edited the description`;
    case 'task.status_changed':
      return `${who} changed status from ${e.old_value} to ${e.new_value}${e.detail ? ` (${e.detail})` : ''}`;
    case 'report.created':
      return `${who} added a report ${q(e.detail)}`;
    case 'report.status_changed':
      return `${who} changed status of ${e.detail || 'a report'} from ${e.old_value} to ${e.new_value}`;
    case 'comment.added':
      return `${who} added a note ${q(e.detail)}`;
    case 'report.comment_added':
      return `${who} added a note on a report ${q(e.detail)}`;
    default:
      return `${who} ${e.action}`;
  }
}
