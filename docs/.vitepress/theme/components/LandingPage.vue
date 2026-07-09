<script setup>
import { withBase } from 'vitepress'

const to = (path) => withBase(path)
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

        <h1 class="hero-title">
          <span class="line"><span>Local</span></span>
          <span class="line"><span class="gradient">LLMs.</span></span>
          <span class="line"><span class="glitch" data-text="Managed.">Managed.</span></span>
        </h1>
        <p class="lede">
          llmctl wraps llama.cpp's <code>llama-server</code> workflow in a terminal UI and a small CLI, so you can
          import models, save reusable profiles, run detached servers, watch health, and get back to work.
        </p>
        <div class="hero-badge">
          <span class="dot"></span>
          Built with Go - Single Binary - Apache 2.0
        </div>
      </div>

      <div class="hero-media">
        <img
          :src="to('/assets/screenshots/landingshowcase.png')"
          alt="llmctl TUI showing models, profiles, running services, and status panes"
          loading="eager"
        />
        <div class="hero-actions">
          <a class="primary-btn" :href="to('/installation')">Installation</a>
          <a class="secondary-btn" :href="to('/quickstart')">Quickstart</a>
          <a class="secondary-btn" href="https://github.com/sockheadrps/llmctl" target="_blank" rel="noopener">GitHub</a>
        </div>
      </div>
    </section>

    <section class="marquee-wrap" aria-label="Highlights">
      <div class="marquee">
        <span>Terminal UI</span><span class="sep">◆</span>
        <span>Profile Management</span><span class="sep">◆</span>
        <span>GGUF Auto-Discovery</span><span class="sep">◆</span>
        <span>Detached Processes</span><span class="sep">◆</span>
        <span>Live Health Monitoring</span><span class="sep">◆</span>
        <span>Token Rate Polling</span><span class="sep">◆</span>
        <span>RPC Offload</span><span class="sep">◆</span>
        <span>Single Binary</span><span class="sep">◆</span>
      </div>
    </section>

    <section class="feature-band">
      <a class="feature-card" :href="to('/quickstart')">
        <span class="feature-kicker">Start here</span>
        <strong>Get a model running fast</strong>
        <span>Follow the shortest path from install to a live local server.</span>
      </a>
      <a class="feature-card" :href="to('/guides/profiles')">
        <span class="feature-kicker">Profiles</span>
        <strong>Save and reuse launch flags</strong>
        <span>Keep fast-draft, high-quality, and custom profiles side by side.</span>
      </a>
      <a class="feature-card" :href="to('/guides/rpc')">
        <span class="feature-kicker">RPC</span>
        <strong>Distribute layers across machines</strong>
        <span>Use the RPC workflow for Linux to Windows GPU offload.</span>
      </a>
      <a class="feature-card" :href="to('/reference/tui')">
        <span class="feature-kicker">Reference</span>
        <strong>Learn the full TUI surface</strong>
        <span>See the tabs, keys, and supported actions in one place.</span>
      </a>
    </section>

    <section class="docs-grid">
      <a class="docs-card" :href="to('/installation')">
        <span>01</span>
        <strong>Install</strong>
        <p>Grab the release artifact, put it on PATH, and launch the TUI.</p>
      </a>
      <a class="docs-card" :href="to('/quickstart')">
        <span>02</span>
        <strong>Quickstart</strong>
        <p>Walk through your first import, profile, and running server.</p>
      </a>
      <a class="docs-card" :href="to('/guides/local-models')">
        <span>03</span>
        <strong>Local models</strong>
        <p>Configure directories and import GGUF files into llmctl.</p>
      </a>
      <a class="docs-card" :href="to('/guides/status-server')">
        <span>04</span>
        <strong>Status server</strong>
        <p>Expose runtime state so other tools can see what is live.</p>
      </a>
      <a class="docs-card" :href="to('/guides/troubleshooting')">
        <span>05</span>
        <strong>Troubleshooting</strong>
        <p>Handle missing binaries, empty model directories, and port issues.</p>
      </a>
      <a class="docs-card" :href="to('/reference/cli')">
        <span>06</span>
        <strong>CLI reference</strong>
        <p>Check the commands and flags without digging through the source.</p>
      </a>
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
  grid-template-columns: .82fr 1.68fr;
  align-items: center;
  padding: 0 0 2rem;
}

.hero-copy {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  width: 100%;
  max-width: none;
}



.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: .5rem;
  margin-top: 1rem;
  padding: .4rem 1rem;
  border-radius: 100px;
  border: 1px solid rgba(168, 85, 247, .3);
  background: rgba(168, 85, 247, .08);
  font-size: .75rem;
  font-weight: 600;
  color: var(--accent);
  animation: badgePulse 3s ease-in-out infinite;
  margin-top: 25px;

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
  line-height: .84;
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
  padding-bottom: .08em;
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

.lede {
  margin: 1.25rem 0 0;
  max-width: 500px;
  font-size: 1rem;
  line-height: 1.75;
  color: rgba(232, 232, 240, 0.78);
}

.hero-actions {
  display: flex;
  gap: .75rem;
  flex-wrap: wrap;
  margin: 2rem 0 2rem;
  align-self: stretch;
  justify-content: flex-end;
  width: max-content;
  margin-left: auto;
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
.secondary-btn:hover,
.feature-card:hover,
.docs-card:hover {
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
  padding: 1rem;
  border-radius: 24px;
  box-shadow: 0 30px 80px rgba(0, 0, 0, .05);
  transform: translateX(.75rem);
}

.hero-media img {
  display: block;
  width: 100%;
  height: auto;
  border-radius: 16px;
}

.feature-band {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1rem;
  margin-top: 1rem;
}

.feature-card,
.docs-card,
.panel,
.quote-panel {
  border: 1px solid var(--line);
  background: var(--surface);
  box-shadow: 0 12px 36px rgba(0, 0, 0, .28);
  backdrop-filter: blur(12px);
}

.feature-card {
  display: flex;
  flex-direction: column;
  gap: .45rem;
  padding: 1rem;
  border-radius: 18px;
  color: inherit;
  text-decoration: none;
  transition: transform .2s ease, border-color .2s ease, background .2s ease;
}

.feature-card:hover {
  border-color: rgba(0, 229, 255, 0.32);
  background: rgba(19, 25, 40, 0.95);
}

.feature-kicker {
  color: var(--accent-2);
}

.feature-card strong,
.docs-card strong {
  color: #f2f6ff;
}

.feature-card span:last-child,
.docs-card p,
.intro-panel p,
.quote-panel blockquote {
  color: rgba(232, 232, 240, 0.72);
  line-height: 1.65;
}

.docs-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  margin-top: 1rem;
}

.docs-card {
  display: flex;
  flex-direction: column;
  gap: .6rem;
  padding: 1.1rem;
  border-radius: 18px;
  text-decoration: none;
  transition: transform .2s ease, border-color .2s ease, background .2s ease;
}

.docs-card:hover {
  border-color: rgba(168, 85, 247, 0.34);
  background: rgba(19, 25, 40, 0.96);
}

.docs-card span {
  color: var(--accent-5);
}

.docs-card p {
  margin: 0;
}

.quote-panel {
  margin-top: 1rem;
  padding: 1.3rem 1.4rem;
  border-radius: 22px;
}

.quote-panel blockquote {
  margin: .35rem 0 0;
  font-size: 1.1rem;
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
  .hero,
  .docs-grid,
  .feature-band {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .landing-shell {
    padding-inline: .75rem;
  }

  .hero {
    padding-top: 1rem;
  }

  .hero-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .primary-btn,
  .secondary-btn {
    width: 100%;
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
</style>
