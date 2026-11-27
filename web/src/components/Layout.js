import { useState } from 'react';
import { Navigate, NavLink, Outlet } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { DataProvider, useData } from '../context/DataContext';
import NewProjectModal from './NewProjectModal';
import Logo from './Logo';

export default function Layout() {
  const { user, loading } = useAuth();
  if (loading) return <div className="page"><p className="muted">loading…</p></div>;
  if (!user) return <Navigate to="/login" replace />;
  return (
    <DataProvider>
      <Shell />
    </DataProvider>
  );
}

const Chevron = ({ open }) => (
  <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true" style={{ transform: open ? 'rotate(90deg)' : 'none' }}>
    <path d="M4 2l4 4-4 4" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
);

function Shell() {
  const { user, logout } = useAuth();
  const { projects, groups, newProjectOpen, setNewProjectOpen } = useData();
  const [projectsOpen, setProjectsOpen] = useState(true);
  const [groupsOpen, setGroupsOpen] = useState(true);
  const cls = ({ isActive }) => `side-link${isActive ? ' active' : ''}`;

  return (
    <div className="shell">
      <aside className="sidebar">
        <NavLink to="/dashboard" className="brand"><Logo size={32} />BOTTY</NavLink>

        <nav className="side-nav">
          <NavLink to="/dashboard" end className={cls}>Dashboard</NavLink>

          {/* Projects dropdown, with + to add one */}
          <div className="side-head">
            <button className="side-toggle" onClick={() => setProjectsOpen((o) => !o)} aria-expanded={projectsOpen}>
              <Chevron open={projectsOpen} /> Projects
              {projects && <span className="count">{projects.length}</span>}
            </button>
            <button className="btn sm icon" title="New project" aria-label="New project" onClick={() => setNewProjectOpen(true)}>+</button>
          </div>
          {projectsOpen && (
            <div className="side-sub">
              {projects === null && <span className="muted">loading…</span>}
              {projects?.length === 0 && <span className="muted">No projects yet</span>}
              {projects?.map((p) => (
                <NavLink key={p.id} to={`/projects/${p.id}`} className={cls}>{p.name}</NavLink>
              ))}
            </div>
          )}

          {/* Groups dropdown */}
          <div className="side-head">
            <button className="side-toggle" onClick={() => setGroupsOpen((o) => !o)} aria-expanded={groupsOpen}>
              <Chevron open={groupsOpen} /> Groups
              {groups && <span className="count">{groups.length}</span>}
            </button>
            <NavLink to="/groups" className="btn sm icon" title="Add the bot to a group" aria-label="Add a group">+</NavLink>
          </div>
          {groupsOpen && (
            <div className="side-sub">
              {groups === null && <span className="muted">loading…</span>}
              {groups?.length === 0 && <span className="muted">No groups yet</span>}
              {groups?.map((g) => (
                <NavLink key={g.id} to="/groups" className="side-link">
                  <span className="ellipsis">{g.name}</span>
                </NavLink>
              ))}
            </div>
          )}
        </nav>

        <div className="side-foot">
          <span className="ellipsis small" title={user.email}>{user.username || user.email}</span>
          <button className="btn sm" onClick={logout}>Log out</button>
        </div>
      </aside>

      <main className="main"><Outlet /></main>
      {newProjectOpen && <NewProjectModal />}
    </div>
  );
}
