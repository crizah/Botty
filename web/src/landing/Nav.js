import { Link } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import Logo from '../components/Logo';

export default function Nav() {
  const { user } = useAuth();
  return (
    <header className="landing-nav">
      <Link to="/" className="landing-nav-mark" aria-label="Home">
        <Logo size={34} />
        <span>Botty</span>
      </Link>

      <nav className="landing-nav-links">
        <a href="#features" className="landing-nav-link">Features</a>
      </nav>

      <div className="landing-nav-actions">
        {user ? (
          <Link to="/dashboard" className="landing-btn landing-btn-primary landing-btn-sm">Dashboard</Link>
        ) : (
          <>
            <Link to="/login" className="landing-nav-link">Sign In</Link>
            <Link to="/signup" className="landing-btn landing-btn-primary landing-btn-sm">Sign Up</Link>
          </>
        )}
      </div>
    </header>
  );
}
