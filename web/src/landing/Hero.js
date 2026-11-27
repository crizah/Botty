import { useEffect, useRef, useState } from 'react';
import { motion, useScroll, useTransform } from 'motion/react';
import PixelBlast from '../components/ui/PixelBlast';
import Logo from '../components/Logo';

// Logo diameter follows the viewport so it stays huge but never overflows.
const logoSize = () => Math.round(Math.min(620, window.innerWidth * 0.86, window.innerHeight * 0.74));

export default function Hero() {
  const heroRef = useRef(null);
  const [size, setSize] = useState(logoSize);
  useEffect(() => {
    const onResize = () => setSize(logoSize());
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, []);

  const { scrollYProgress } = useScroll({ target: heroRef, offset: ['start start', 'end start'] });
  const bgOpacity = useTransform(scrollYProgress, [0, 0.55], [0.35, 0]);
  const contentOpacity = useTransform(scrollYProgress, [0, 0.5], [1, 0]);
  const contentY = useTransform(scrollYProgress, [0, 0.5], [0, -40]);

  return (
    <section className="landing-hero" ref={heroRef}>
      <motion.div className="landing-hero-bg" style={{ opacity: bgOpacity }}>
        <PixelBlast
          variant="square" pixelSize={4} color="#d4d4d8" patternScale={2} patternDensity={0.25}
          pixelSizeJitter={0} enableRipples rippleSpeed={0.4} rippleThickness={0.12}
          rippleIntensityScale={1.5} liquid={false} speed={0.5} edgeFade={0.15} transparent
        />
      </motion.div>

      <motion.div className="landing-hero-content" style={{ opacity: contentOpacity, y: contentY }}>
        <Logo size={size} />
      </motion.div>

      <motion.a href="#features" className="landing-hero-scroll" style={{ opacity: contentOpacity }}>
        scroll ↓
      </motion.a>
    </section>
  );
}
