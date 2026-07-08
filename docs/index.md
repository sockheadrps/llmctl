---
layout: home
---

<div class="hero-split">
  <div class="hero-text">
    <p class="hero-name">llmctl</p>
    <h1 class="hero-tagline">Local LLMs. Managed.</h1>
    <p class="hero-sub">Run, configure, and distribute llama.cpp models from your terminal — no config files, no flags, no babysitting.</p>
    <div class="hero-actions">
      <a class="hero-btn-alt" href="/installation">Installation</a>
      <a class="hero-btn-alt" href="https://github.com/sockheadrps/llmctl" target="_blank" rel="noopener">GitHub ↗</a>
    </div>
  </div>
  <div class="hero-media">
    <img
      src="/assets/screenshots/landingshowcase.png"
      alt="llmctl TUI — active services, live GPU telemetry, and RPC backend status"
      class="hero-img"
      loading="eager"
    />

  </div>
</div>

<div class="features-wrap">
  <div class="features-grid">
    <a class="feature-card" href="/quickstart">
      <span class="feature-card-title">Quickstart</span>
      <span class="feature-card-sub">Get a model serving in minutes</span>
    </a>
    <a class="feature-card" href="/guides/local-models">
      <span class="feature-card-title">Local Inference</span>
      <span class="feature-card-sub">GPU, CPU, or mixed — switch instantly</span>
    </a>
    <a class="feature-card" href="/guides/rpc">
      <span class="feature-card-title">Distributed RPC</span>
      <span class="feature-card-sub">Pool GPU layers across machines</span>
    </a>
    <a class="feature-card" href="/guides/benchmarking">
      <span class="feature-card-title">Benchmarking</span>
      <span class="feature-card-sub">Compare tok/s across models and quants</span>
    </a>
  </div>
</div>

<div class="about-wrap">
  <div class="about-inner">
    <h2>What is llmctl?</h2>
    <p>
      llmctl is a terminal UI for managing local LLM inference. It wraps
      <a href="https://github.com/ggerganov/llama.cpp">llama.cpp</a>'s
      <code>llama-server</code> process — handling startup, configuration, health monitoring, and teardown — so you can
      focus on running models rather than managing processes.
    </p>
    <p>
      Everything happens inside the TUI. Import models, create profiles, start instances, and watch them in real time
      without touching a config file or memorizing command-line flags.
    </p>
    <div class="about-features">
      <div class="about-feature">
        <strong>Profile system</strong>
        <span>Define multiple named configurations per model (ports, GPU layers, context sizes) and switch between them instantly.</span>
      </div>
      <div class="about-feature">
        <strong>RPC distribution</strong>
        <span>Split a model's layers across two GPUs on separate machines, increasing effective context size and throughput.</span>
      </div>
      <div class="about-feature">
        <strong>Live telemetry</strong>
        <span>VRAM usage, tok/s current/average/peak, and load times tracked per instance and persisted across sessions.</span>
      </div>
      <div class="about-feature">
        <strong>Status server</strong>
        <span>Expose runtime state over HTTP so other llmctl instances or external tools can monitor what's running.</span>
      </div>
    </div>
  </div>
</div>

<style>
/* ── Hero split ──────────────────────────────────────────────────────────── */
.hero-split {
  display: flex;
  align-items: center;
  max-width: 1400px;
  margin: 0 auto;
  padding: 2rem 1rem 1rem;
}

.hero-text {
  flex: 0 0 320px;
}

.hero-name {
  margin: 0 0 .25rem;
  font-size: 1rem;
  font-weight: 600;
  background: linear-gradient(120deg, #bd34fe 30%, #41d1ff);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  letter-spacing: .02em;
}

.hero-tagline {
  margin: 0 0 1rem;
  font-size: 3rem;
  font-weight: 700;
  line-height: 1.12;
  color: var(--vp-c-text-1);
}

.hero-sub {
  margin: 0 0 2rem;
  font-size: .9375rem;
  color: var(--vp-c-text-2);
  line-height: 1.65;
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: .625rem;
}

.hero-btn-brand,
.hero-btn-alt {
  display: inline-block;
  padding: .55rem 1.4rem;
  border-radius: 8px;
  font-size: .875rem;
  font-weight: 500;
  text-decoration: none;
  transition: border-color .2s, background .2s, box-shadow .2s, color .2s;
}

.hero-btn-brand {
  background: var(--vp-c-brand-1, #3dd68c);
  color: #000;
}

.hero-btn-brand:hover {
  background: var(--vp-c-brand-2, #5ee0a0);
}

.hero-btn-alt {
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  box-shadow: 0 1px 0 rgba(255,255,255,0.08) inset, 0 2px 8px rgba(0,0,0,0.2);
  color: var(--vp-c-text-1);
}

.hero-btn-alt:hover {
  border-color: rgba(65, 209, 255, 0.45);
  background: rgba(65, 209, 255, 0.07);
  box-shadow: 0 1px 0 rgba(255,255,255,0.1) inset, 0 2px 12px rgba(65,209,255,0.12);
  color: #41d1ff;
}

.hero-media {
  flex: 1;
  min-width: 0;
  text-align: center;
}

.hero-img {
  display: block;
  width: 100%;
  height: auto;
  border-radius: 10px;
}

.hero-caption {
  margin: .625rem 0 0;
  font-size: .8125rem;
  color: var(--vp-c-text-3);
}

@media (max-width: 768px) {
  .hero-split {
    flex-direction: column;
    padding: 3rem 1.5rem 2.5rem;
    text-align: center;
  }
  .hero-text { flex: none; }
  .hero-actions { justify-content: center; }
}

/* ── Feature cards ───────────────────────────────────────────────────────── */
.features-wrap {
  padding: 1.5rem 2.5rem 3rem;
}

.features-grid {
  max-width: 1280px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1rem;
}

@media (max-width: 768px) {
  .features-grid { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 480px) {
  .features-grid { grid-template-columns: 1fr; }
}

.feature-card {
  display: flex;
  flex-direction: column;
  gap: .25rem;
  padding: .875rem 1.1rem;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.04);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  box-shadow: 0 1px 0 0 rgba(255,255,255,0.06) inset, 0 4px 16px rgba(0,0,0,0.25);
  text-decoration: none;
  color: inherit;
  transition: border-color .2s, background .2s, box-shadow .2s;
}

.feature-card:hover {
  border-color: rgba(65, 209, 255, 0.35);
  background: rgba(65, 209, 255, 0.04);
  box-shadow: 0 1px 0 0 rgba(255,255,255,0.08) inset, 0 4px 24px rgba(65,209,255,0.08);
}

.feature-card:hover .feature-card-title {
  color: #41d1ff;
}

.feature-card-title {
  font-size: .875rem;
  font-weight: 600;
  color: var(--vp-c-text-1);
  transition: color .2s;
}

.feature-card-sub {
  font-size: .775rem;
  color: var(--vp-c-text-2);
  line-height: 1.4;
}

/* ── About section ───────────────────────────────────────────────────────── */
.about-wrap {
  padding: 4rem 1.5rem;
}

.about-inner {
  max-width: 720px;
  margin: 0 auto;
}

.about-inner h2 {
  font-size: 1.5rem;
  font-weight: 700;
  margin: 0 0 1rem;
  color: var(--vp-c-text-1);
}

.about-inner > p {
  color: var(--vp-c-text-2);
  line-height: 1.75;
  margin: 0 0 1rem;
}

.about-inner a {
  color: var(--vp-c-brand-1, #3dd68c);
  text-decoration: none;
}

.about-inner a:hover { text-decoration: underline; }

.about-features {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1.25rem;
  margin-top: 2.5rem;
}

@media (max-width: 560px) {
  .about-features { grid-template-columns: 1fr; }
}

.about-feature {
  display: flex;
  flex-direction: column;
  gap: .35rem;
}

.about-feature strong {
  font-size: .9rem;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.about-feature span {
  font-size: .85rem;
  color: var(--vp-c-text-2);
  line-height: 1.55;
}
</style>
