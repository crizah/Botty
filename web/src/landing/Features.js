import ChatShowcase from './showcases/ChatShowcase';
import { SCENARIOS } from './showcases/scenarios';

// Copy is used verbatim from the brief (first letter capitalised only).
// `body` lines marked TODO: change are stubs.
const FEATURES = [
  {
    key: 'link',
    kicker: 'Linking',
    title: 'Link a group chat in one message',
    // TODO: change — stub copy
    body: 'Add Botty to a WhatsApp group, then tag it with your project link code. Botty connects the group to that project, and every task from that chat lands on its board.',
    points: [],
  },
  {
    key: 'create',
    kicker: 'Create',
    title: 'Tag Botty, get a ticket',
    // TODO: change — stub intro; the bullet is verbatim
    body: 'See an issue in the chat? Tag Botty and describe it. The task shows up on the dashboard before the conversation moves on.',
    points: ['Uses a decision model to decide intent, grouping and duplication of existing tasks, therefore, dirt cheap ;)'],
  },
  {
    key: 'close',
    kicker: 'Resolve',
    title: 'Close the loop in the chat',
    body: '',
    points: [
      'Get tasks created and closed without leaving the whatsapp interface, botty takes care of it',
      'Botty supports a full audit of every state change on the task, who complained, who reported it, who resolved it etc',
    ],
  },
  {
    key: 'dupes',
    kicker: 'Grouping',
    title: 'Same problem, one task',
    body: '',
    points: [
      'Duplicate problems are grouped together as a reports on a task, for your....convenience',
      'Botty only reads messages its tagged in *cough* meta *cough*',
      'Botty understands direct instructions or reply tags, where it walks the chain on replies to get to the real issue',
    ],
  },
  {
    key: 'hinglish',
    kicker: 'Hinglish & images',
    title: 'Speak how you speak',
    body: '',
    points: [
      'It supports Hinglish!!!',
      'Supports adding images directly to the task dashboard when captioned with a complaint',
    ],
  },
  {
    key: 'guard',
    kicker: 'Good bot',
    title: 'Only the tasks that matter',
    body: '',
    points: [
      'Botty ignores non complaine/task requests, when tagged and otherwise. good bot :3',
      'Supports allow lists for group chats so untrusted EVIL entities dont overwhelm botty with spam',
    ],
  },
];

export default function Features() {
  return (
    <section id="features" className="fscroll">
      <div className="fscroll-intro">
        <span className="fscroll-heading-tag">What Botty does</span>
        <h2>One bot, <em>zero lost tasks</em></h2>
      </div>

      <div className="fscroll-list">
        {FEATURES.map((f, i) => (
          <div className={`fscroll-row${i % 2 ? ' reverse' : ''}`} key={f.key}>
            <div className="fscroll-text">
              <span className="fscroll-kicker">{f.kicker}</span>
              <h3>{f.title}</h3>
              {f.body && <p>{f.body}</p>}
              {f.points.length > 0 && (
                <ul className="fscroll-points">
                  {f.points.map((p) => <li key={p}>{p}</li>)}
                </ul>
              )}
            </div>
            <div className="fscroll-showcase">
              <ChatShowcase scenario={SCENARIOS[f.key]} />
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
