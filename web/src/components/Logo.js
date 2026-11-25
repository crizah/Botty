import CircularText from './ui/CircularText';
import './Logo.css';

// Renders the circular text at a fixed 120px, then scales the whole layer down
// to `size` to avoid sub-pixel jitter from rotating tiny text.
export default function Logo({ size = 44 }) {
  const scale = size / 120;
  return (
    <span className="botty-logo" style={{ width: size, height: size }}>
      <div className="botty-logo-scale" style={{ transform: `translate(-50%, -50%) scale(${scale})` }}>
        <CircularText text="BOTTY*BOTTY*" onHover="speedUp" spinDuration={20} className="botty-logo-circular" />
      </div>
    </span>
  );
}
