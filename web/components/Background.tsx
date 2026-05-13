import { SharkScene } from "./SharkScene";

// Background — atmospheric backdrop. Layered, fixed to the viewport,
// rendered behind everything. Most layers are server-renderable; the
// SharkScene canvas is "use client" because it needs requestAnimationFrame.
//
// Layer stack (back → front):
//   1. Radial gradient base wash.
//   2. Two drifting aurora blobs (brand cyan + deeper cyan).
//   3. SharkScene — dot-grid + autonomous shark with repulsion.
//   4. SVG noise overlay (grain texture).
//   5. Edge vignette to deepen the corners.

export function Background() {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none fixed inset-0 -z-10 overflow-hidden"
    >
      {/* 1. Base wash */}
      <div
        className="absolute inset-0"
        style={{
          background:
            "radial-gradient(ellipse 90% 70% at 50% 0%, #0A1626 0%, #050A10 60%)",
        }}
      />

      {/* 2a. Aurora blob — bright cyan, top-left */}
      <div
        className="absolute -left-[10vw] -top-[15vw] h-[55vw] w-[55vw] rounded-full opacity-[0.55] mix-blend-screen blur-[80px] animate-drift-a"
        style={{
          background:
            "radial-gradient(circle at 30% 30%, #00ACDE 0%, rgba(0,172,222,0.4) 30%, transparent 70%)",
        }}
      />

      {/* 2b. Aurora blob — deeper cyan, bottom-right */}
      <div
        className="absolute -bottom-[15vw] -right-[10vw] h-[55vw] w-[55vw] rounded-full opacity-[0.45] mix-blend-screen blur-[90px] animate-drift-b"
        style={{
          background:
            "radial-gradient(circle at 70% 70%, #007DA8 0%, rgba(0,125,168,0.35) 30%, transparent 70%)",
        }}
      />

      {/* 3. Dot-grid + shark */}
      <SharkScene />

      {/* 4. Noise overlay — SVG turbulence; 4% opacity, overlay blend. */}
      <svg
        className="absolute inset-0 h-full w-full opacity-[0.04] mix-blend-overlay"
        xmlns="http://www.w3.org/2000/svg"
      >
        <filter id="grain">
          <feTurbulence
            type="fractalNoise"
            baseFrequency="0.85"
            numOctaves="2"
            stitchTiles="stitch"
          />
          <feColorMatrix type="saturate" values="0" />
        </filter>
        <rect width="100%" height="100%" filter="url(#grain)" />
      </svg>

      {/* 5. Edge vignette */}
      <div
        className="absolute inset-0"
        style={{
          background:
            "radial-gradient(ellipse 80% 60% at 50% 50%, transparent 60%, rgba(5,10,16,0.6) 100%)",
        }}
      />
    </div>
  );
}
