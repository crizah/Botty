import { Link } from 'react-router-dom';
import Logo from '../components/Logo';

export default function Footer() {
  return (
    <footer className="landing-footer">
      <Logo size={64} />
      {/* TODO: change — footer tagline is a stub */}
      <p className="landing-footer-tagline">Turn group chats into tracked tasks.</p>
      <div className="landing-footer-actions">
        <Link to="/login" className="landing-btn landing-btn-ghost">Sign In</Link>
        <Link to="/signup" className="landing-btn landing-btn-primary">Sign Up</Link>
      </div>
      <p className="landing-footer-copy">© {new Date().getFullYear()} Botty. All rights reserved.</p>
    </footer>
  );
}
