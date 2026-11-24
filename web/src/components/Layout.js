import { Link, NavLink, Navigate, Outlet } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

export default function Layout() {
  const { user, loading, logout } = useAuth();
  if (loading) return <div className="page"><p className="muted">loading…</p></div>;
  if (!user) return <Navigate to="/login" replace />;

  return (
    <>
      <nav className="nav">
        <Link to="/" className="brand">BOTTY</Link>
        <NavLink to="/" end className={({ isActive }) => `link${isActive ? ' active' : ''}`}>Projects</NavLink>
        <NavLink to="/groups" className={({ isActive }) => `link${isActive ? ' active' : ''}`}>Groups</NavLink>
        <span className="spacer" />
        <span className="small">{user.username || user.email}</span>
        <button className="btn sm" onClick={logout}>Log out</button>
      </nav>
      <Outlet />
    </>
  );
}
