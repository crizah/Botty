import PixelBlast from './ui/PixelBlast';
import Logo from './Logo';
import './AuthShell.css';

// Split-screen auth layout: animated panel on the left, form on the right.
export default function AuthShell({ title, subtitle, children, footer }) {
  return (
    <div className="auth-root">
      <div className="auth-brand-wrapper">
        <div className="auth-brand-panel">
          <div style={{ position: 'absolute', inset: 0, width: '100%', height: '100%' }}>
            <PixelBlast
              variant="square"
              pixelSize={4}
              color="#d4d4d8"
              patternScale={2}
              patternDensity={1}
              pixelSizeJitter={0}
              enableRipples
              rippleSpeed={0.4}
              rippleThickness={0.12}
              rippleIntensityScale={1.5}
              liquid={false}
              speed={0.5}
              edgeFade={0.25}
              transparent
            />
          </div>
        </div>
      </div>

      <div className="auth-form-panel">
        <div className="auth-form-container">
          <div className="auth-head">
            <Logo size={44} />
            <h1>{title}</h1>
            {subtitle && <p className="auth-sub">{subtitle}</p>}
          </div>
          {children}
          {footer && <p className="auth-foot">{footer}</p>}
        </div>
      </div>
    </div>
  );
}
