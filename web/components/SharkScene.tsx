"use client";

import { useEffect, useRef } from "react";
import { toast } from "sonner";

// SharkScene draws a dot-grid background that is perturbed by a slow-
// swimming shark. Everything happens in a single 2D canvas:
//
//   - Dot grid:  base positions, soft cyan, ~1.4px radius.
//   - Shark:     autonomous steering toward random targets; turns
//                smoothly via short-arc angle interpolation.
//   - Repulsion: each dot is pushed away from the shark with a
//                squared falloff, then springs back to its base
//                position. The effect is a "bow wave" trailing the
//                shark as it cuts through the grid.
//
// The component honors prefers-reduced-motion (renders one static
// frame instead of animating) and pauses when the tab is hidden.

interface Dot {
  baseX: number;
  baseY: number;
  x: number;
  y: number;
}

interface Shark {
  x: number;
  y: number;
  vx: number;
  vy: number;
  tx: number;
  ty: number;
  heading: number;
  bodyPhase: number;
  spinStart: number; // ms timestamp of last click; drives the barrel-roll
  hovered: boolean;
}

interface Particle {
  x: number;
  y: number;
  vx: number;
  vy: number;
  life: number; // ms elapsed
  maxLife: number;
}

// Hit radius around shark center, in CSS px. Roughly half the body
// length at SHARK_SCALE — generous so the egg is easy to find.
const HIT_RADIUS = 80;
const SPIN_DURATION = 800; // ms — full 360° barrel roll
const PARTICLE_COUNT = 16;

// Easter-egg messages — shark gym-bro energy. Picked at random on each
// click, with no immediate repeat so consecutive clicks feel fresh.
const EGG_MESSAGES = [
  "🦈 Hey — easy on the snout.",
  "🦈 Don't skip fin day.",
  "🦈 Megalodon bulk went entirely to the jaws.",
  "🦈 Can't stop swimming. Can't stop grinding.",
  "🦈 Trying to hit a PR but I have no bones, just cartilage.",
  "🦈 Pectoral fin pump went crazy today.",
  "🦈 Whale sharks eat plankton just for the volume.",
  "🦈 Remora fish — just standing there, waiting for me to finish my set.",
  "🦈 Bro, do you even migrate?",
  "🦈 Is salt water bad for my creatine bloating?",
  "🦈 Hammerheads using the mirror to look at two squat racks at once.",
  "🦈 Nurse sharks sleeping on the gym floor between sets again.",
  "🦈 Tracking macros when your entire diet is raw seal.",
  "🦈 The absolute panic when a Great White asks to bench press swim-in.",
  "🦈 Swimming laps around the coral reef just to flex on the anemones.",
  "🦈 You found me a lot. That's commitment.",
];

const CELL = 36; // grid spacing in CSS px
const DOT_RADIUS = 1.4;
const SHARK_SCALE = 1.5; // 1.5x bigger than the base path geometry
const REPULSE = 195; // px — radius of influence, scaled with the shark
const MAX_PUSH = 32; // px — max displacement at zero distance
const MAX_SPEED = 1.4; // px/frame at 60fps
const STEER = 0.038;
const FRICTION = 0.965;
const TURN_RATE = 0.09;

function shortestAngleDelta(target: number, current: number): number {
  let d = target - current;
  while (d > Math.PI) d -= 2 * Math.PI;
  while (d < -Math.PI) d += 2 * Math.PI;
  return d;
}

function drawShark(ctx: CanvasRenderingContext2D, phase: number, hovered: boolean) {
  // Stroke uses a pale cyan; fill is the brand with low alpha so the
  // shark reads as a silhouette rather than a flat blob. Hover state
  // brightens both and adds a subtle outer glow.
  const stroke = hovered ? "rgba(230, 248, 254, 1)" : "rgba(184, 230, 247, 0.85)";
  const fill = hovered ? "rgba(0, 172, 222, 0.32)" : "rgba(0, 172, 222, 0.18)";
  if (hovered) {
    ctx.shadowColor = "rgba(0, 172, 222, 0.8)";
    ctx.shadowBlur = 18;
  } else {
    ctx.shadowBlur = 0;
  }

  // Subtle vertical wag on the tail tip — purely cosmetic, period 1.2s.
  const wag = Math.sin(phase) * 4;

  ctx.lineJoin = "round";
  ctx.lineCap = "round";
  ctx.strokeStyle = stroke;
  ctx.fillStyle = fill;
  ctx.lineWidth = 1;

  // BODY — sleek torpedo, snout at +x, peduncle at -50.
  ctx.beginPath();
  ctx.moveTo(56, 0);
  ctx.bezierCurveTo(40, -8, 8, -13, -45, -4);
  ctx.lineTo(-50, 0);
  ctx.bezierCurveTo(8, 13, 40, 8, 56, 0);
  ctx.closePath();
  ctx.fill();
  ctx.stroke();

  // DORSAL FIN — triangular, slightly back of body midpoint.
  ctx.beginPath();
  ctx.moveTo(-6, -9);
  ctx.lineTo(2, -24);
  ctx.lineTo(14, -9);
  ctx.closePath();
  ctx.fill();
  ctx.stroke();

  // PECTORAL FIN — underside, swept back, gives speed cues.
  ctx.beginPath();
  ctx.moveTo(10, 5);
  ctx.lineTo(-2, 17);
  ctx.lineTo(22, 8);
  ctx.closePath();
  ctx.fill();
  ctx.stroke();

  // CAUDAL FIN — asymmetric crescent, top lobe longer (great-white shape).
  ctx.beginPath();
  ctx.moveTo(-50, 0);
  ctx.lineTo(-74, -18 + wag);
  ctx.lineTo(-58, -2);
  ctx.lineTo(-58, 2);
  ctx.lineTo(-66, 12 + wag);
  ctx.closePath();
  ctx.fill();
  ctx.stroke();

  // GILL SLITS — three fine strokes.
  ctx.lineWidth = 0.7;
  for (let i = 0; i < 3; i++) {
    ctx.beginPath();
    ctx.moveTo(14 + i * 4, -5);
    ctx.lineTo(14 + i * 4, 5);
    ctx.stroke();
  }
  ctx.lineWidth = 1;

  // EYE — small white dot with darker pupil.
  ctx.beginPath();
  ctx.arc(38, -3, 1.5, 0, Math.PI * 2);
  ctx.fillStyle = "rgba(220, 240, 252, 0.9)";
  ctx.fill();
  ctx.beginPath();
  ctx.arc(38, -3, 0.7, 0, Math.PI * 2);
  ctx.fillStyle = "rgba(5, 10, 16, 1)";
  ctx.fill();
}

export function SharkScene() {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    let raf = 0;
    let running = true;
    let dots: Dot[] = [];

    // Two sharks, swimming independently. Each holds its own steering
    // state; the dot grid feels the combined repulsion.
    const sharks: Shark[] = [
      { x: 0, y: 0, vx: 0, vy: 0, tx: 0, ty: 0, heading: 0, bodyPhase: 0, spinStart: 0, hovered: false },
      { x: 0, y: 0, vx: 0, vy: 0, tx: 0, ty: 0, heading: Math.PI, bodyPhase: 1.7, spinStart: 0, hovered: false },
    ];

    // Easter-egg state.
    let particles: Particle[] = [];
    let mouseX = -1;
    let mouseY = -1;
    let lastEggIdx = -1;

    function viewport() {
      return { w: window.innerWidth, h: window.innerHeight };
    }

    function pickTarget(s: Shark) {
      const { w, h } = viewport();
      const margin = 100;
      s.tx = margin + Math.random() * Math.max(w - 2 * margin, 100);
      s.ty = margin + Math.random() * Math.max(h - 2 * margin, 100);
    }

    function build() {
      const { w, h } = viewport();
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      canvas!.width = Math.floor(w * dpr);
      canvas!.height = Math.floor(h * dpr);
      canvas!.style.width = `${w}px`;
      canvas!.style.height = `${h}px`;
      ctx!.setTransform(dpr, 0, 0, dpr, 0, 0);

      // Inset the grid by half a cell so it doesn't kiss the edges.
      dots = [];
      const offX = CELL / 2 + ((w % CELL) / 2);
      const offY = CELL / 2 + ((h % CELL) / 2);
      for (let y = offY; y < h - offY / 2; y += CELL) {
        for (let x = offX; x < w - offX / 2; x += CELL) {
          dots.push({ baseX: x, baseY: y, x, y });
        }
      }

      // First-time shark placement — opposite corners, opposite targets,
      // so they cross paths organically rather than tracking each other.
      if (sharks[0].x === 0 && sharks[0].y === 0) {
        sharks[0].x = w * 0.2;
        sharks[0].y = h * 0.5;
        sharks[0].tx = w * 0.75;
        sharks[0].ty = h * 0.35;
        sharks[0].heading = 0;

        sharks[1].x = w * 0.8;
        sharks[1].y = h * 0.55;
        sharks[1].tx = w * 0.25;
        sharks[1].ty = h * 0.7;
        sharks[1].heading = Math.PI;
      }
    }

    function hitShark(sh: Shark, mx: number, my: number) {
      const dx = mx - sh.x;
      const dy = my - sh.y;
      return dx * dx + dy * dy < HIT_RADIUS * HIT_RADIUS;
    }

    function pickEggMessage(): string {
      // Random selection, but avoid the immediate-previous index so
      // consecutive clicks never show the same line twice.
      let idx = Math.floor(Math.random() * EGG_MESSAGES.length);
      if (idx === lastEggIdx && EGG_MESSAGES.length > 1) {
        idx = (idx + 1) % EGG_MESSAGES.length;
      }
      lastEggIdx = idx;
      return EGG_MESSAGES[idx];
    }

    function triggerEgg(sh: Shark) {
      sh.spinStart = performance.now();
      // Burst of cyan particles from the shark's current center.
      for (let i = 0; i < PARTICLE_COUNT; i++) {
        const angle = Math.random() * Math.PI * 2;
        const speed = 1.6 + Math.random() * 1.8;
        particles.push({
          x: sh.x,
          y: sh.y,
          vx: Math.cos(angle) * speed,
          vy: Math.sin(angle) * speed,
          life: 0,
          maxLife: 700 + Math.random() * 500,
        });
      }
      toast.success(pickEggMessage(), { duration: 3200 });
    }

    function step() {
      const { w, h } = viewport();

      // STEER each shark toward its own target.
      for (let s = 0; s < sharks.length; s++) {
        const sh = sharks[s];
        const dx = sh.tx - sh.x;
        const dy = sh.ty - sh.y;
        const dist = Math.hypot(dx, dy) || 0.01;
        if (dist < 70) pickTarget(sh);

        sh.vx = (sh.vx + (dx / dist) * STEER) * FRICTION;
        sh.vy = (sh.vy + (dy / dist) * STEER) * FRICTION;

        const speed = Math.hypot(sh.vx, sh.vy);
        if (speed > MAX_SPEED) {
          sh.vx = (sh.vx / speed) * MAX_SPEED;
          sh.vy = (sh.vy / speed) * MAX_SPEED;
        }
        sh.x += sh.vx;
        sh.y += sh.vy;

        // SOFT WALL — if a shark wanders off-screen, pick a new target.
        if (sh.x < 40 || sh.x > w - 40 || sh.y < 40 || sh.y > h - 40) {
          pickTarget(sh);
        }

        // SMOOTH HEADING — short-arc interpolation toward velocity vector.
        if (speed > 0.05) {
          const desired = Math.atan2(sh.vy, sh.vx);
          sh.heading += shortestAngleDelta(desired, sh.heading) * TURN_RATE;
        }
        sh.bodyPhase += 0.12;
      }

      // HOVER state per shark + cursor. Body cursor is overridden when
      // a shark is under the pointer; UI elements (buttons, inputs) keep
      // their own cursor since they specify it explicitly.
      let anyHover = false;
      for (let s = 0; s < sharks.length; s++) {
        const sh = sharks[s];
        sh.hovered = mouseX >= 0 && hitShark(sh, mouseX, mouseY);
        if (sh.hovered) anyHover = true;
      }
      document.body.style.cursor = anyHover ? "pointer" : "";

      // PARTICLES — apply friction, advance life, drop dead ones.
      for (let i = 0; i < particles.length; i++) {
        const p = particles[i];
        p.x += p.vx;
        p.y += p.vy;
        p.vx *= 0.96;
        p.vy *= 0.96;
        p.life += 16;
      }
      if (particles.length > 0) {
        particles = particles.filter((p) => p.life < p.maxLife);
      }

      // UPDATE dots — combined repulsion from every shark.
      for (let i = 0; i < dots.length; i++) {
        const d = dots[i];
        let pushX = 0;
        let pushY = 0;
        for (let s = 0; s < sharks.length; s++) {
          const sh = sharks[s];
          const ex = d.baseX - sh.x;
          const ey = d.baseY - sh.y;
          const er = Math.hypot(ex, ey) || 0.01;
          if (er < REPULSE) {
            const f = (1 - er / REPULSE) ** 2;
            pushX += (ex / er) * f * MAX_PUSH;
            pushY += (ey / er) * f * MAX_PUSH;
          }
        }
        const tx = d.baseX + pushX;
        const ty = d.baseY + pushY;
        d.x += (tx - d.x) * 0.22;
        d.y += (ty - d.y) * 0.22;
      }

      draw();
      if (running) raf = requestAnimationFrame(step);
    }

    function draw() {
      const { w, h } = viewport();
      ctx!.clearRect(0, 0, w, h);

      // DOTS — single fillStyle pass; tiny circles render faster than rects.
      ctx!.fillStyle = "rgba(0, 172, 222, 0.36)";
      for (let i = 0; i < dots.length; i++) {
        const d = dots[i];
        ctx!.beginPath();
        ctx!.arc(d.x, d.y, DOT_RADIUS, 0, Math.PI * 2);
        ctx!.fill();
      }

      // PARTICLES — fading cyan dots from the easter-egg burst.
      if (particles.length > 0) {
        ctx!.fillStyle = "rgba(0, 172, 222, 1)";
        for (let i = 0; i < particles.length; i++) {
          const p = particles[i];
          const t = 1 - p.life / p.maxLife;
          ctx!.globalAlpha = t * 0.85;
          ctx!.beginPath();
          ctx!.arc(p.x, p.y, 2.5 * t + 0.5, 0, Math.PI * 2);
          ctx!.fill();
        }
        ctx!.globalAlpha = 1;
      }

      // SHARKS — translate, rotate, scale 1.5x, draw. Per-shark. The
      // optional spin offset gives the click-easter-egg its barrel roll.
      const now = performance.now();
      for (let s = 0; s < sharks.length; s++) {
        const sh = sharks[s];
        let spinAngle = 0;
        if (sh.spinStart > 0) {
          const elapsed = now - sh.spinStart;
          if (elapsed < SPIN_DURATION) {
            // Ease-out spin for a satisfying flip-and-settle feel.
            const t = elapsed / SPIN_DURATION;
            spinAngle = 2 * Math.PI * (1 - (1 - t) ** 3);
          } else {
            sh.spinStart = 0;
          }
        }
        ctx!.save();
        ctx!.translate(sh.x, sh.y);
        ctx!.rotate(sh.heading + spinAngle);
        ctx!.scale(SHARK_SCALE, SHARK_SCALE);
        drawShark(ctx!, sh.bodyPhase, sh.hovered);
        ctx!.restore();
      }
    }

    const onResize = () => build();
    const onVisibility = () => {
      if (document.hidden) {
        running = false;
        cancelAnimationFrame(raf);
      } else if (!running) {
        running = true;
        raf = requestAnimationFrame(step);
      }
    };

    // Mouse position is tracked at the WINDOW level so the easter egg
    // works no matter what DOM element is currently under the pointer.
    // The canvas is full-viewport and uses fixed positioning, so
    // viewport coords (clientX/clientY) match canvas coords directly.
    const onMove = (e: MouseEvent) => {
      mouseX = e.clientX;
      mouseY = e.clientY;
    };
    const onLeave = () => {
      mouseX = -1;
      mouseY = -1;
    };
    const onClick = (e: MouseEvent) => {
      for (let s = 0; s < sharks.length; s++) {
        if (hitShark(sharks[s], e.clientX, e.clientY)) {
          triggerEgg(sharks[s]);
          return;
        }
      }
    };

    build();

    const reduced =
      typeof window !== "undefined" &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches;

    if (reduced) {
      // Static frame: sharks resting on opposite sides, dots at base.
      const { w, h } = viewport();
      sharks[0].x = w * 0.3;
      sharks[0].y = h * 0.45;
      sharks[0].heading = 0.15;
      sharks[1].x = w * 0.72;
      sharks[1].y = h * 0.55;
      sharks[1].heading = Math.PI - 0.2;
      draw();
    } else {
      raf = requestAnimationFrame(step);
    }

    window.addEventListener("resize", onResize);
    document.addEventListener("visibilitychange", onVisibility);
    window.addEventListener("mousemove", onMove);
    window.addEventListener("click", onClick);
    document.addEventListener("mouseleave", onLeave);

    return () => {
      running = false;
      cancelAnimationFrame(raf);
      window.removeEventListener("resize", onResize);
      document.removeEventListener("visibilitychange", onVisibility);
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("click", onClick);
      document.removeEventListener("mouseleave", onLeave);
      // Restore any cursor override we applied on hover.
      document.body.style.cursor = "";
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      aria-hidden="true"
      // Hit-testing for the easter egg is done at the window level so
      // it works regardless of which element is on top — the canvas
      // itself stays pointer-events-none to avoid stealing selection
      // or input focus from the cards.
      className="absolute inset-0 h-full w-full"
    />
  );
}
