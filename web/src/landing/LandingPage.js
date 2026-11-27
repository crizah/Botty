import Nav from './Nav';
import Hero from './Hero';
import About from './About';
import Features from './Features';
import Footer from './Footer';
import './landing.css';

export default function LandingPage() {
  return (
    <div className="landing-page">
      <Nav />
      <Hero />
      <About />
      <Features />
      <Footer />
    </div>
  );
}
