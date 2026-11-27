import Nav from './Nav';
import Hero from './Hero';
import Features from './Features';
import Footer from './Footer';
import './landing.css';

export default function LandingPage() {
  return (
    <div className="landing-page">
      <Nav />
      <Hero />
      <Features />
      <Footer />
    </div>
  );
}
