<script setup>
import { withBase } from 'vitepress'
import { onBeforeUnmount, onMounted, ref } from 'vue'
import dashboardShot from '../../../assets/screenshots/dashboard.png'
import landingShot from '../../../assets/screenshots/newlanding.png'

const to = (path) => withBase(path)
const activeHero = ref('landing')
const heroCards = [
  {
    id: 'landing',
    src: landingShot,
    alt: 'llmctl TUI showing models, profiles, running services, and status panes',
  },
  {
    id: 'dashboard',
    src: dashboardShot,
    alt: 'llmctl dashboard showing active models, source trends, and GPU utilization',
  },
]

const angle = 12
const remap = (value, oldMax, newMax) => {
  const newValue = ((value + oldMax) * (newMax * 2)) / (oldMax * 2) - newMax
  return Math.min(Math.max(newValue, -newMax), newMax)
}

const toggleHeroFront = (heroId) => {
  if (activeHero.value === heroId) {
    activeHero.value = heroId === 'landing' ? 'dashboard' : 'landing'
    return
  }

  activeHero.value = heroId
}

const setCardPose = (cardEl, rotateX, rotateY) => {
  cardEl.style.setProperty('--rotate-x', `${rotateX}deg`)
  cardEl.style.setProperty('--rotate-y', `${rotateY}deg`)
  cardEl.style.setProperty('--lift', `${Math.abs(rotateX) * 0.08 + Math.abs(rotateY) * 0.06}px`)
  cardEl.style.setProperty('--glow-x', `${50 + rotateY * 1.4}%`)
  cardEl.style.setProperty('--glow-y', `${50 - rotateX * 1.4}%`)
}

const resetCardPose = (cardEl) => {
  cardEl.style.setProperty('--rotate-x', '0deg')
  cardEl.style.setProperty('--rotate-y', '0deg')
  cardEl.style.setProperty('--lift', '0px')
  cardEl.style.setProperty('--glow-x', '50%')
  cardEl.style.setProperty('--glow-y', '50%')
}

const handleCardMove = (heroId, event) => {
  if (stopIntroMotion) {
    stopIntroMotion()
    stopIntroMotion = null
  }

  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    return
  }

  const cardEl = event.currentTarget?.closest('.hero-stack-card')
  if (!cardEl) {
    return
  }

  if (activeHero.value !== heroId) {
    return
  }

  const rect = cardEl.getBoundingClientRect()
  const x = event.clientX - (rect.left + rect.width / 2)
  const y = event.clientY - (rect.top + rect.height / 2)
  setCardPose(cardEl, remap(y, rect.height / 2, angle) * -1, remap(x, rect.width / 2, angle))
}

const handleCardLeave = (heroId, event) => {
  const cardEl = event.currentTarget?.closest('.hero-stack-card')
  if (!cardEl) {
    return
  }

  resetCardPose(cardEl)

  if (activeHero.value === heroId) {
    if (stopIntroMotion) {
      stopIntroMotion()
    }
    stopIntroMotion = startIntroMotion()
  }
}

const getFrontCardFrame = () => document.querySelector('.hero-stack-card.is-front .hero-card-frame')

const startIntroMotion = () => {
  const frame = getFrontCardFrame()
  if (!frame || window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    return () => {}
  }

  let frameId = null
  let stopped = false
  let start = 0
  const duration = 1400
  const radius = 6
  const frequency = 2.2

  const tick = (now) => {
    if (stopped) {
      return
    }

    if (!start) {
      start = now
    }

    const progress = Math.min((now - start) / duration, 1)
    const eased = 1 - Math.pow(1 - progress, 3)
    const angle = now / 1000 * frequency * Math.PI * 2
    const rotateX = Math.sin(angle) * radius * (1 - eased)
    const rotateY = Math.cos(angle) * radius * (1 - eased)

    setCardPose(frame.closest('.hero-stack-card'), rotateX, rotateY)

    if (progress < 1) {
      frameId = window.requestAnimationFrame(tick)
      return
    }

    resetCardPose(frame.closest('.hero-stack-card'))
  }

  frameId = window.requestAnimationFrame(tick)

  const stop = () => {
    stopped = true
    if (frameId !== null) {
      window.cancelAnimationFrame(frameId)
    }
    resetCardPose(frame.closest('.hero-stack-card'))
  }

  const stopOnUser = () => stop()
  frame.addEventListener('pointermove', stopOnUser, { once: true, passive: true })
  frame.addEventListener('pointerdown', stopOnUser, { once: true })
  frame.addEventListener('click', stopOnUser, { once: true })
  frame.addEventListener('pointerleave', stopOnUser, { once: true })

  return stop
}

let stopIntroMotion = null
let cleanupHero = null

onMounted(() => {
  const observer = new IntersectionObserver((entries) => {
    entries.forEach(e => {
      if (e.isIntersecting) {
        e.target.classList.add('visible')
        observer.unobserve(e.target)
      }
    })
  }, { threshold: .15, rootMargin: '0px 0px -40px 0px' })

  document.querySelectorAll('.reveal').forEach(el => observer.observe(el))

  stopIntroMotion = startIntroMotion()
  cleanupHero = () => {
    if (stopIntroMotion) {
      stopIntroMotion()
      stopIntroMotion = null
    }
    observer.disconnect()
  }
})

onBeforeUnmount(() => {
  if (cleanupHero) cleanupHero()
})
</script>

<template>
  <div class="landing-shell">
    <div class="bg-grid"></div>
    <div class="orb orb-1"></div>
    <div class="orb orb-2"></div>
    <div class="orb orb-3"></div>
    <div class="noise"></div>
    <section class="hero">
      <div class="hero-copy">
        <div class="hero-badge">
          <span class="dot"></span>
          Go · Single Binary · Apache 2.0
        </div>
        <h1 class="hero-title">
          <span class="line"><span>Local</span></span>
          <span class="line"><span class="gradient">LLMs.</span></span>
          <span class="line"><span class="glitch" data-text="Managed.">Managed.</span></span>
        </h1>
        <p class="lede">
          Import models, save profiles, run detached servers — all from one terminal UI or a single CLI command.
        </p>
        <div class="hero-actions">
          <a class="primary-btn" :href="to('/installation')">Installation</a>
          <a class="secondary-btn" :href="to('/quickstart')">Quickstart</a>
          <a class="secondary-btn" href="https://github.com/sockheadrps/llmctl" target="_blank" rel="noopener">GitHub</a>
        </div>
      </div>

      <div
        class="hero-media hero-stack"
        :style="{ '--hero-url': `url(${landingShot})` }"
      >
        <button
          v-for="(card, index) in heroCards"
          :key="card.id"
          type="button"
          class="hero-stack-card"
          :data-hero-id="card.id"
          :class="{
            'is-front': activeHero === card.id,
            'is-back': activeHero !== card.id,
            'is-landscape': index === 0,
            'is-dashboard': index === 1,
          }"
          :aria-pressed="activeHero === card.id"
          :tabindex="0"
        >
          <div class="hero-card-shadow"></div>
          <div
            class="hero-card-frame"
            @pointermove="(event) => handleCardMove(card.id, event)"
            @pointerleave="(event) => handleCardLeave(card.id, event)"
            @pointercancel="(event) => handleCardLeave(card.id, event)"
            @click="toggleHeroFront(card.id)"
          >
            <div class="hero-card-glow"></div>
            <img
              :src="to(card.src)"
              :alt="card.alt"
              loading="eager"
              class="hero-card-image"
            />
            <div class="hero-card-sheen" aria-hidden="true"></div>
          </div>
        </button>
      </div>

      <div class="marquee-wrap" aria-label="Highlights">
        <div class="marquee">
          <span>Terminal UI</span><span class="sep">◆</span>
          <span>Profile Management</span><span class="sep">◆</span>
          <span>GGUF Auto-Discovery</span><span class="sep">◆</span>
          <span>Detached Processes</span><span class="sep">◆</span>
          <span>Live Health Monitoring</span><span class="sep">◆</span>
          <span>Token Rate Polling</span><span class="sep">◆</span>
          <span>RPC Offload</span><span class="sep">◆</span>
          <span>Single Binary</span><span class="sep">◆</span>
          <span>Terminal UI</span><span class="sep">◆</span>
          <span>Profile Management</span><span class="sep">◆</span>
          <span>GGUF Auto-Discovery</span><span class="sep">◆</span>
          <span>Detached Processes</span><span class="sep">◆</span>
          <span>Live Health Monitoring</span><span class="sep">◆</span>
          <span>Token Rate Polling</span><span class="sep">◆</span>
          <span>RPC Offload</span><span class="sep">◆</span>
          <span>Single Binary</span><span class="sep">◆</span>
        </div>
      </div>

      <div class="hero-scroll">
        <span>Scroll</span>
        <div class="scroll-line"></div>
      </div>
    </section>

    <section class="cards-section">
      <div class="section-label reveal">Get Around</div>
      <h2 class="section-title reveal reveal-delay-1">
        Start anywhere.<br>
        <span style="color:var(--text2)">Everything is one click away.</span>
      </h2>
      <p class="section-subtitle reveal reveal-delay-2">
        Jump to the workflow you need — first run, profile setup, RPC offload, or the full TUI reference.
      </p>

      <div class="cards-shell">
        <div class="cards-grid cards-grid-features">
          <a class="feature-card reveal" :href="to('/quickstart')">
            <span class="feature-kicker">Start here</span>
            <strong>Get a model running fast</strong>
            <span>Follow the shortest path from install to a live local server.</span>
          </a>
          <a class="feature-card reveal reveal-delay-1" :href="to('/guides/profiles')">
            <span class="feature-kicker">Profiles</span>
            <strong>Save and reuse launch flags</strong>
            <span>Keep fast-draft, high-quality, and custom profiles side by side.</span>
          </a>
          <a class="feature-card reveal reveal-delay-2" :href="to('/guides/rpc')">
            <span class="feature-kicker">RPC</span>
            <strong>Distribute layers across machines</strong>
            <span>Use the RPC workflow for Linux to Windows GPU offload.</span>
          </a>
          <a class="feature-card reveal reveal-delay-3" :href="to('/reference/tui')">
            <span class="feature-kicker">Reference</span>
            <strong>Learn the full TUI surface</strong>
            <span>See the tabs, keys, and supported actions in one place.</span>
          </a>
        </div>

        <div class="cards-grid cards-grid-docs">
          <a class="docs-card reveal" :href="to('/installation')">
            <span>01</span>
            <strong>Install</strong>
            <p>Grab the release artifact, put it on PATH, and launch the TUI.</p>
          </a>
          <a class="docs-card reveal reveal-delay-1" :href="to('/quickstart')">
            <span>02</span>
            <strong>Quickstart</strong>
            <p>Walk through your first import, profile, and running server.</p>
          </a>
          <a class="docs-card reveal reveal-delay-2" :href="to('/guides/local-models')">
            <span>03</span>
            <strong>Local models</strong>
            <p>Configure directories and import GGUF files into llmctl.</p>
          </a>
          <a class="docs-card reveal reveal-delay-3" :href="to('/guides/status-server')">
            <span>04</span>
            <strong>Status server</strong>
            <p>Expose runtime state so other tools can see what is live.</p>
          </a>
          <a class="docs-card reveal reveal-delay-4" :href="to('/guides/troubleshooting')">
            <span>05</span>
            <strong>Troubleshooting</strong>
            <p>Handle missing binaries, empty model directories, and port issues.</p>
          </a>
          <a class="docs-card reveal reveal-delay-4" :href="to('/reference/cli')">
            <span>06</span>
            <strong>CLI reference</strong>
            <p>Check the commands and flags without digging through the source.</p>
          </a>
        </div>
      </div>
    </section>

    <section class="quote-panel">
      <p class="section-label">The idea</p>
      <blockquote>
        llmctl is the layer between "I know what I want to run" and "I have to remember that command again."
      </blockquote>
    </section>
  </div>
</template>

<style scoped>
:global(:root) {
  --surface: rgba(11, 13, 22, 0.82);
  --surface-2: rgba(16, 20, 32, 0.92);
  --line: rgba(255, 255, 255, 0.09);
  --accent: #00e5ff;
  --accent-2: #06d6a0;
  --accent-3: #a855f7;
  --accent-4: #ff3e6c;
  --accent-5: #ffd166;
}

.landing-shell {
  position: relative;
  isolation: isolate;
  max-width: 1180px;
  margin: 0 auto;
  padding: 1rem 1rem 4rem;
}

.bg-grid,
.noise,
.orb {
  pointer-events: none;
  position: fixed;
  inset: 0;
  z-index: -2;
}

.bg-grid {
  z-index: -3;
  opacity: 0.38;
  background-image:
    linear-gradient(rgba(168, 85, 247, 0.08) 1px, transparent 1px),
    linear-gradient(90deg, rgba(168, 85, 247, 0.08) 1px, transparent 1px);
  background-size: 84px 84px;
  animation: gridMove 30s linear infinite;
}

.noise {
  z-index: -1;
  opacity: 0.04;
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='.85' numOctaves='4' stitchTiles='yes'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
  background-size: 256px 256px;
}

.orb {
  position: fixed;
  z-index: -2;
  border-radius: 50%;
  filter: blur(120px);
  opacity: 0.38;
}

.orb-1 {
  width: 560px;
  height: 560px;
  top: -180px;
  left: -160px;
  background: var(--accent-3);
  animation: orbFloat1 14s ease-in-out infinite;
}

.orb-2 {
  width: 480px;
  height: 480px;
  right: -140px;
  top: 18%;
  background: var(--accent);
  animation: orbFloat2 18s ease-in-out infinite;
}

.orb-3 {
  width: 420px;
  height: 420px;
  left: 12%;
  bottom: -120px;
  background: var(--accent-4);
  animation: orbFloat3 20s ease-in-out infinite;
}

.hero {
  display: grid;
  grid-template-columns: minmax(0, 0.98fr) minmax(0, 1.72fr);
  grid-template-rows: 1fr auto;
  align-items: center;
  gap: 3rem 3.25rem;
  min-height: calc(100vh - var(--vp-nav-height));
  position: relative;
}


.hero-copy  { grid-column: 1; grid-row: 1; }
.hero-media {
  grid-column: 2;
  grid-row: 1;
  justify-self: end;
  width: min(100%, 720px);
}
.marquee-wrap { grid-column: 1 / -1; grid-row: 2; align-self: end; }

.hero-scroll {
  position: absolute;
  bottom: 1.5rem;
  left: 50%;
  transform: translateX(-50%) translateY(60px);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: .5rem;
  color: rgba(232, 232, 240, 0.4);
  font-size: .68rem;
  letter-spacing: .2em;
  text-transform: uppercase;
  animation: scrollBounce 2s ease-in-out infinite;
  pointer-events: none;
  bottom: 8rem;
}

.scroll-line {
  width: 1px;
  height: 40px;
  background: linear-gradient(to bottom, var(--accent-3), transparent);
  animation: scrollLine 2s ease-in-out infinite;
}

@keyframes scrollBounce {
  0%, 100% { transform: translateX(-50%) translateY(0); }
  50%       { transform: translateX(-50%) translateY(8px); }
}

@keyframes scrollLine {
  0%   { opacity: 0; transform: scaleY(0); transform-origin: top; }
  50%  { opacity: 1; transform: scaleY(1); }
  100% { opacity: 0; transform: scaleY(0); transform-origin: bottom; }
}

.hero-copy {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  width: 100%;
}



.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: .5rem;
  padding: .4rem 1rem;
  border-radius: 100px;
  border: 1px solid rgba(168, 85, 247, .3);
  background: rgba(168, 85, 247, .08);
  font-size: .75rem;
  font-weight: 600;
  color: var(--accent);
  animation: badgePulse 3s ease-in-out infinite;
  margin-bottom: 1.25rem;
}

.hero-badge .dot {
  width: 6px;
  height: 6px;
  background: var(--accent-2);
  border-radius: 50%;
  animation: dotBlink 2s ease-in-out infinite;
}

.hero-title {
  margin: 0;
  font-size: clamp(2.8rem, 7.2vw, 5.7rem);
  font-weight: 900;
  line-height: .94;
  letter-spacing: -.04em;
  margin-bottom: .9rem;
  position: relative;
  color: #f2f6ff;
}

.hero-title .line {
  display: block;
  overflow: hidden;
  padding-bottom: 0;
}

.hero-title .line span {
  display: inline-block;
  transform: translateY(110%);
  animation: lineReveal .8s cubic-bezier(.77, 0, .18, 1) forwards;
}

.hero-title .line:nth-child(2) span {
  animation-delay: .15s;
  
}

.hero-title .line:nth-child(3) span {
  animation-delay: .3s;
}

.hero-title .gradient {
  background: linear-gradient(135deg, var(--accent-3), var(--accent), var(--accent-2), var(--accent-5));
  background-size: 300% 300%;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  animation: gradientShift 6s ease-in-out infinite;
}

.hero-title .glitch {
  position: relative;
  display: inline-block;
}

.hero-title .glitch::before,
.hero-title .glitch::after {
  content: attr(data-text);
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  -webkit-text-fill-color: initial;
}

.hero-title .glitch::before {
  color: var(--accent-3);
  animation: glitch1 3s infinite;
  clip-path: inset(0 0 60% 0);
  -webkit-text-fill-color: var(--accent-3);
  opacity: .8;
}

.hero-title .glitch::after {
  color: var(--accent);
  animation: glitch2 3s infinite;
  clip-path: inset(60% 0 0 0);
  -webkit-text-fill-color: var(--accent);
  opacity: .8;
}

.hero-title .line:last-child {
  overflow: visible;
  padding-bottom: .24em;
}

.eyebrow,
.section-label,
.feature-kicker,
.docs-card span {
  font-size: .74rem;
  font-weight: 700;
  letter-spacing: .18em;
  text-transform: uppercase;
}

.eyebrow,
.section-label {
  color: var(--accent);
  margin: 0 0 .85rem;
}

.section-label {
  display: flex;
  align-items: center;
  gap: .75rem;
}

.section-label::before {
  content: '';
  width: 30px;
  height: 1px;
  background: var(--accent);
}

.section-title {
  font-size: clamp(2rem, 5vw, 3.5rem);
  font-weight: 900;
  line-height: 1.1;
  margin: 0 0 1.5rem;
  letter-spacing: -.03em;
}

.section-subtitle {
  font-size: 1.15rem;
  color: rgba(232, 232, 240, 0.72);
  max-width: 600px;
  line-height: 1.7;
  margin: 0;
}

.lede {
  max-width: 500px;
  font-size: 1rem;
  line-height: 1.75;
  color: rgba(232, 232, 240, 0.78);
  opacity: 0;
  transform: translateY(16px);
  animation: ledeReveal 1.1s cubic-bezier(.4, 0, .2, 1) .9s forwards;
}

@keyframes ledeReveal {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.hero-actions {
  display: flex;
  gap: .75rem;
  flex-wrap: wrap;
  margin-top: 2rem;
}

.primary-btn,
.secondary-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 42px;
  padding: .7rem 1rem;
  border-radius: 999px;
  text-decoration: none;
  font-weight: 600;
  transition: transform .2s ease, border-color .2s ease, background .2s ease, color .2s ease;
}

.primary-btn {
  color: #08111a;
  background: linear-gradient(135deg, var(--accent), var(--accent-2));
  box-shadow: 0 6px 26px rgba(0, 229, 255, 0.24);
}

.secondary-btn {
  color: #f2f6ff;
  border: 1px solid var(--line);
  background: rgba(255, 255, 255, 0.04);
  backdrop-filter: blur(12px);
}

.primary-btn:hover,
.secondary-btn:hover {
  transform: translateY(-2px);
}

.hero-pills {
  display: flex;
  flex-wrap: wrap;
  gap: .5rem;
  margin-top: 1.5rem;
}

.hero-pills span {
  padding: .4rem .7rem;
  border: 1px solid var(--line);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.04);
  color: rgba(232, 232, 240, 0.72);
  font-size: .85rem;
}

.hero-media {
  box-shadow: 0 30px 80px rgba(0, 0, 0, .05);
}

.hero-stack {
  position: relative;
  min-height: clamp(340px, 48vw, 720px);
  perspective: 1400px;
  transform-style: preserve-3d;
  isolation: isolate;
  overflow: visible;
}

.hero-stack-card {
  position: absolute;
  inset: 0;
  display: block;
  width: 100%;
  padding: 0;
  border: 0;
  border-radius: 28px;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
  appearance: none;
  transform-style: preserve-3d;
  pointer-events: none;
  transition:
    transform .48s cubic-bezier(.2, .85, .25, 1),
    opacity .28s ease,
    filter .28s ease;
  will-change: transform;
}

.hero-stack-card:focus-visible {
  outline: 2px solid rgba(0, 229, 255, 0.8);
  outline-offset: 8px;
}

.hero-stack-card.is-landscape.is-front {
  z-index: 2;
  transform:
    translate3d(0, 0, 0)
    rotateX(var(--rotate-y, 0deg))
    rotateY(var(--rotate-x, 0deg))
    scale(1);
}

.hero-stack-card.is-dashboard.is-front {
  z-index: 2;
  transform:
    translate3d(0, 0, 0)
    rotateX(var(--rotate-y, 0deg))
    rotateY(var(--rotate-x, 0deg))
    scale(1);
}

.hero-stack-card.is-landscape.is-back {
  z-index: 1;
  transform: translate3d(124px, 98px, -78px) scale(0.86);
  opacity: 0.84;
  filter: saturate(0.9) brightness(0.92);
}

.hero-stack-card.is-dashboard.is-back {
  z-index: 1;
  transform: translate3d(182px, -90px, -70px) scale(0.65);
  opacity: 0.9;
  filter: saturate(0.92) brightness(0.94);
}

.hero-stack-card.is-front:hover .hero-card-frame {
  box-shadow:
    0 28px 72px rgba(0, 0, 0, 0.36),
    0 0 0 1px rgba(255, 255, 255, 0.09) inset;
}

.hero-card-shadow {
  position: absolute;
  inset: 62px;
  border-radius: 28px;
  background:
    radial-gradient(circle at 50% 55%, rgba(168, 85, 247, 0.16), transparent 56%),
    radial-gradient(circle at 62% 34%, rgba(0, 229, 255, 0.12), transparent 36%),
    var(--hero-url);
  background-size: cover;
  background-position: center;
  filter: blur(15px) saturate(0.98);
  opacity: 0.05;
}

.hero-stack-card.is-back .hero-card-shadow {
  opacity: 0.08;
  filter: blur(7px) saturate(0.8);
  transform: translate3d(0, 8px, -8px) scale(0.98);
}

.hero-card-frame {
  position: relative;
  border-radius: 28px;
  overflow: hidden;
  pointer-events: auto;
  cursor: pointer;
  transform-style: preserve-3d;
  transform:
    perspective(1400px)
    rotateX(var(--rotate-y, 0deg))
    rotateY(var(--rotate-x, 0deg))
    translate3d(0, 0, 0);
  transition: transform .18s ease, box-shadow .18s ease;
  box-shadow:
    0 24px 60px rgba(0, 0, 0, 0.32),
    0 0 0 1px rgba(255, 255, 255, 0.06) inset;
}

.hero-card-image {
  display: block;
  width: 100%;
  height: auto;
  transform: translate3d(0, 0, 1px);
  border-radius: 28px;
  pointer-events: auto;
}

.hero-card-glow,
.hero-card-sheen {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.hero-card-glow {
  background:
    radial-gradient(circle at var(--glow-x, 50%) var(--glow-y, 50%), rgba(255, 255, 255, 0.14), transparent 30%),
    linear-gradient(135deg, rgba(168, 85, 247, 0.04), rgba(0, 229, 255, 0.015));
  mix-blend-mode: screen;
  opacity: 0.15;
}

.hero-stack-card.is-back .hero-card-glow {
  opacity: 0.05;
}

.hero-card-sheen {
  background:
    linear-gradient(120deg, transparent 0%, rgba(255, 255, 255, 0.18) 48%, transparent 52%);
  transform: translateX(-34%) translateY(-10%) rotate(8deg);
  opacity: 0.10;
  mix-blend-mode: screen;
}

.quote-panel {
  margin-top: 3rem;
  padding: 1.5rem 1.75rem;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, .06);
  background: rgba(255, 255, 255, .02);
  border-left: 3px solid var(--accent-3);
}

.quote-panel blockquote {
  margin: .35rem 0 0;
  font-size: 1.1rem;
  color: rgba(232, 232, 240, .85);
  font-style: italic;
  line-height: 1.65;
}

.marquee-wrap {
  padding: 1.1rem 0;
  overflow: hidden;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.marquee {
  display: flex;
  gap: 1.6rem;
  width: max-content;
  animation: marqueeScroll 22s linear infinite;
}

.marquee span {
  display: flex;
  align-items: center;
  gap: 1.6rem;
  white-space: nowrap;
  font-size: .8rem;
  letter-spacing: .12em;
  text-transform: uppercase;
  color: rgba(232, 232, 240, 0.72);
}

.marquee .sep {
  color: var(--accent);
  opacity: .5;
}

@media (max-width: 960px) {
  .hero {
    grid-template-columns: 1fr;
  }

  .hero-copy  { grid-column: 1; grid-row: 1; }
  .hero-media { grid-column: 1; grid-row: 2; }
  .marquee-wrap { grid-column: 1; grid-row: 3; }
}

@media (max-width: 720px) {
  .landing-shell {
    padding-inline: .75rem;
  }

  .hero {
    padding-top: 1rem;
  }

  .hero-actions {
    flex-wrap: wrap;
  }

  .hero-stack {
    transform: none !important;
    perspective: none;
  }

  .hero-stack {
    min-height: 0;
  }

  .hero-stack-card {
    position: relative;
    inset: auto;
    margin-bottom: 1rem;
    transform: none !important;
    opacity: 1 !important;
    filter: none !important;
  }

  .hero-card-frame {
    transform: none;
  }

  .hero-card-shadow {
    inset: 10px;
    filter: blur(18px);
  }

  .hero-card-image {
    border-radius: 24px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .hero-stack,
  .hero-card-frame,
  .hero-card-shadow,
  .hero-card-glow,
  .hero-card-sheen {
    animation: none !important;
    transition: none !important;
  }

  .hero-card-frame {
    transform: none;
  }
}

@keyframes gridMove {
  0% { transform: translateY(0); }
  100% { transform: translateY(84px); }
}

@keyframes badgePulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(168, 85, 247, .2); }
  50% { box-shadow: 0 0 0 8px rgba(168, 85, 247, 0); }
}

@keyframes dotBlink {
  0%, 100% { opacity: 1; }
  50% { opacity: .3; }
}

@keyframes lineReveal {
  to { transform: translateY(0); }
}

@keyframes gradientShift {
  0% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
  100% { background-position: 0% 50%; }
}

@keyframes glitch1 {
  0%, 90%, 100% { transform: translate(0); }
  92% { transform: translate(-4px, 2px); }
  94% { transform: translate(4px, -1px); }
  96% { transform: translate(-2px, 3px); }
}

@keyframes glitch2 {
  0%, 90%, 100% { transform: translate(0); }
  91% { transform: translate(3px, -2px); }
  93% { transform: translate(-3px, 1px); }
  95% { transform: translate(2px, -3px); }
}

@keyframes orbFloat1 {
  0%, 100% { transform: translate(0, 0) scale(1); }
  50% { transform: translate(90px, 40px) scale(1.06); }
}

@keyframes orbFloat2 {
  0%, 100% { transform: translate(0, 0) scale(1); }
  50% { transform: translate(-80px, 60px) scale(1.08); }
}

@keyframes orbFloat3 {
  0%, 100% { transform: translate(0, 0) scale(1); }
  50% { transform: translate(50px, -30px) scale(0.96); }
}

@keyframes marqueeScroll {
  0% { transform: translateX(0); }
  100% { transform: translateX(-50%); }
}

/* ── Scroll Reveal ── */
.reveal {
  opacity: 0;
  transform: translateY(48px);
  transition: opacity 1s cubic-bezier(.16, 1, .3, 1), transform 1s cubic-bezier(.16, 1, .3, 1);
}

.reveal.visible {
  opacity: 1;
  transform: translateY(0);
}

.reveal-delay-1 { transition-delay: .12s; }
.reveal-delay-2 { transition-delay: .26s; }
.reveal-delay-3 { transition-delay: .4s; }
.reveal-delay-4 { transition-delay: .54s; }

/* marquee fades in without the upward slide */
.marquee-wrap.reveal {
  transform: none;
  transition: opacity 1.2s ease;
}

/* ── Cards Section ── */
.cards-section {
  padding: 6rem 0 3rem;
  position: relative;
}

.cards-shell {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  margin-top: 3rem;
}

.cards-grid {
  display: grid;
  gap: 1.5rem;
}

.cards-grid-features {
  grid-template-columns: repeat(2, 1fr);
}

.cards-grid-docs {
  grid-template-columns: repeat(3, 1fr);
}

@media (max-width: 960px) {
  .cards-grid-features {
    grid-template-columns: 1fr;
  }

  .cards-grid-docs {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 540px) {
  .cards-grid-features,
  .cards-grid-docs {
    grid-template-columns: 1fr;
  }
}

/* ── Feature & Docs Cards — t1.html problem-card style ── */
.feature-card,
.docs-card {
  padding: 2rem;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, .06);
  background: rgba(255, 255, 255, .02);
  position: relative;
  overflow: hidden;
  text-decoration: none;
  color: inherit;
  display: flex;
  flex-direction: column;
  gap: .5rem;
  transition: border-color .4s, transform .4s, box-shadow .4s;
}

.feature-card:hover,
.docs-card:hover {
  border-color: rgba(168, 85, 247, .3);
  transform: translateY(-5px);
  box-shadow: 0 20px 60px rgba(0, 0, 0, .3);
}

.feature-card::before,
.docs-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(90deg, var(--accent-3), var(--accent-4), var(--accent));
  opacity: 0;
  transition: opacity .4s;
}

.feature-card:hover::before,
.docs-card:hover::before {
  opacity: 1;
}

.feature-kicker {
  color: var(--accent-3);
}

.feature-card strong {
  font-size: 1rem;
  font-weight: 700;
  color: #f2f6ff;
}

.feature-card > span:not(.feature-kicker) {
  font-size: .85rem;
  color: rgba(232, 232, 240, .72);
  line-height: 1.6;
}

.docs-card > span {
  font-family: 'JetBrains Mono', monospace;
  color: var(--accent);
  margin-bottom: .25rem;
}

.docs-card strong {
  font-size: 1rem;
  font-weight: 700;
  color: #f2f6ff;
}

.docs-card p {
  font-size: .85rem;
  color: rgba(232, 232, 240, .72);
  line-height: 1.6;
  margin: 0;
}
</style>
