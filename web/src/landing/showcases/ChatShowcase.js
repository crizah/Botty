import { useEffect, useRef, useState } from 'react';
import { AnimatePresence, motion, useInView } from 'motion/react';
import './showcase.css';

const HOLD = 2600; // how long the finished state stays before looping

// Drives the script: reveals one message at a time, optionally pans to the
// dashboard, holds, then resets. Only runs while scrolled into view.
function useScript(scenario, active) {
  const [count, setCount] = useState(0);
  const [panned, setPanned] = useState(false);

  useEffect(() => {
    if (!active) return undefined;
    let timer;
    const msgs = scenario.messages;
    const step = (i) => {
      if (i < msgs.length) {
        timer = setTimeout(() => { setCount(i + 1); step(i + 1); }, msgs[i].delay);
      } else if (scenario.pan && i === msgs.length) {
        timer = setTimeout(() => { setPanned(true); step(i + 1); }, 1100);
      } else {
        timer = setTimeout(() => { setCount(0); setPanned(false); step(0); }, HOLD + (scenario.pan ? 1500 : 0));
      }
    };
    setCount(0); setPanned(false); step(0);
    return () => clearTimeout(timer);
  }, [scenario, active]);

  return { count, panned };
}

const WhatsAppGlyph = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" aria-hidden="true">
    <path fill="#25d366" d="M12 2a10 10 0 0 0-8.6 15.1L2 22l5-1.3A10 10 0 1 0 12 2z" />
    <path fill="#fff" d="M8.6 7.3c-.2-.4-.4-.4-.6-.4h-.5c-.2 0-.5.1-.7.3-.2.3-.9.9-.9 2.2s.9 2.5 1 2.7c.1.2 1.8 2.8 4.4 3.8 2.2.9 2.6.7 3.1.6.5 0 1.5-.6 1.7-1.2.2-.6.2-1.1.2-1.2-.1-.1-.2-.2-.5-.3l-1.600-.8c-.2-.1-.4-.1-.6.1l-.7.9c-.1.2-.3.2-.5.1-.3-.1-1.100-.4-2-1.300-.7-.7-1.200-1.500-1.300-1.800-.1-.2 0-.4.1-.5l.4-.4c.1-.1.2-.3.2-.4.100-.2 0-.3 0-.4z" />
  </svg>
);

function Mentions({ text }) {
  return text.split(/(@botty)/g).map((part, i) =>
    part === '@botty' ? <span key={i} className="wa-mention">{part}</span> : part
  );
}

function Washer() {
  return (
    <div className="wa-photo" aria-label="photo of a washing machine">
      <svg viewBox="0 0 80 80" width="100%" height="100%">
        <rect x="14" y="8" width="52" height="66" rx="6" fill="#2a3942" stroke="#4b5b64" strokeWidth="2" />
        <circle cx="40" cy="46" r="18" fill="#111b21" stroke="#6b7c85" strokeWidth="3" />
        <circle cx="40" cy="46" r="10" fill="#1f2c34" />
        <rect x="20" y="14" width="24" height="6" rx="3" fill="#4b5b64" />
        <circle cx="54" cy="17" r="3" fill="#6b7c85" />
      </svg>
    </div>
  );
}

function Bubble({ m, parent }) {
  return (
    <motion.div
      className={`wa-row ${m.bot ? 'bot' : ''}`}
      initial={{ opacity: 0, y: 10, scale: 0.96 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ type: 'spring', damping: 24, stiffness: 320 }}
    >
      <div className={`wa-bubble${m.bot ? ' bot' : ''}`}>
        <span className={`wa-name ${m.bot ? 'bot' : ''}`}>{m.bot ? 'Botty' : m.from}</span>
        {parent && (
          <div className="wa-quote">
            <span className="wa-quote-name">{parent.bot ? 'Botty' : parent.from}</span>
            <span className="wa-quote-text">{parent.text}</span>
          </div>
        )}
        {m.image === 'washer' && <Washer />}
        <span className="wa-text"><Mentions text={m.text} /></span>
      </div>
    </motion.div>
  );
}

function Chat({ scenario, count }) {
  const msgs = scenario.messages.slice(0, count);
  const byId = Object.fromEntries(scenario.messages.map((m) => [m.id, m]));
  const bodyRef = useRef(null);
  useEffect(() => {
    if (bodyRef.current) bodyRef.current.scrollTop = bodyRef.current.scrollHeight;
  }, [count]);

  return (
    <>
      <div className="wa-bar">
        <span className="wa-avatar" />
        <div className="wa-bar-title">
          <strong>{scenario.group}</strong>
          <span>Asha, Dev, Kabir, Botty</span>
        </div>
        <span className="wa-brand"><WhatsAppGlyph /> WhatsApp</span>
      </div>
      <div className="wa-body" ref={bodyRef}>
        {msgs.map((m) => <Bubble key={m.id} m={m} parent={m.replyTo ? byId[m.replyTo] : null} />)}
      </div>
    </>
  );
}

function TaskCard({ t, fresh }) {
  return (
    <motion.div
      className={`dash-card${fresh ? ' fresh' : ''}`}
      initial={fresh ? { opacity: 0, y: -12, scale: 0.96 } : false}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ delay: fresh ? 0.45 : 0, type: 'spring', damping: 22, stiffness: 260 }}
    >
      <div className="dash-card-top">
        <span className="dash-ref">{t.ref}</span>
        <span className={`dash-tag ${t.status}`}>{t.status}</span>
      </div>
      <div className="dash-title">{t.title}</div>
      {t.image === 'washer' && <div className="dash-thumb"><Washer /></div>}
      {t.reports > 0 && (
        <div className="dash-reports">
          <div className="dash-progress">{Array.from({ length: t.reports }).map((_, i) => <span key={i} />)}</div>
          <span>{t.reports} {t.reports === 1 ? 'report' : 'reports'}</span>
        </div>
      )}
    </motion.div>
  );
}

function Dashboard({ pan }) {
  const open = [pan.added, ...pan.tasks.filter((t) => t.status === 'open')];
  const done = pan.tasks.filter((t) => t.status === 'done');
  return (
    <div className="dash">
      <div className="dash-head"><strong>Tasks</strong></div>
      <div className="dash-cols">
        <div className="dash-col">
          <div className="dash-col-head">Open <span className="dash-count">{open.length}</span></div>
          {open.map((t, i) => <TaskCard key={t.ref} t={t} fresh={i === 0} />)}
        </div>
        <div className="dash-col">
          <div className="dash-col-head">Done <span className="dash-count">{done.length}</span></div>
          {done.map((t) => <TaskCard key={t.ref} t={t} />)}
        </div>
      </div>
    </div>
  );
}

export default function ChatShowcase({ scenario }) {
  const ref = useRef(null);
  const inView = useInView(ref, { margin: '-15% 0px' });
  const { count, panned } = useScript(scenario, inView);

  return (
    <div className="sc-window" ref={ref}>
      <div className="sc-chrome">
        <span className="sc-dot" /><span className="sc-dot" /><span className="sc-dot" />
        <span className="sc-path">{panned ? 'botty.app/projects/hostel' : 'web.whatsapp.com'}</span>
      </div>
      <div className="sc-stage">
        <AnimatePresence initial={false} mode="popLayout">
          {!panned ? (
            <motion.div key="chat" className="sc-pane wa" initial={{ x: 0 }} exit={{ x: '-100%' }} transition={{ duration: 0.55, ease: [0.16, 1, 0.3, 1] }}>
              <Chat scenario={scenario} count={count} />
            </motion.div>
          ) : (
            <motion.div key="dash" className="sc-pane" initial={{ x: '100%' }} animate={{ x: 0 }} transition={{ duration: 0.55, ease: [0.16, 1, 0.3, 1] }}>
              <Dashboard pan={scenario.pan} />
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
