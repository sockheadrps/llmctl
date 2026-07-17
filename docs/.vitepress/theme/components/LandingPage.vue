<script setup>
import { withBase } from 'vitepress'
import { onBeforeUnmount, onMounted, ref } from 'vue'

const to = (path) => withBase(path)
const activeHero = ref('landing')
const marqueeWrap = ref(null)
const terminalLines = [
  'llama-server \\',
  '  -m models Ternary-Bonsai-27B-Q2_0.gguf \\',
  '  -md models Ternary-Bonsai-27B-dspark-Q4_1.gguf \\',
  '  --spec-type draft-dspark \\',
  '  --spec-draft-n-max 4 \\',
  '  -ngl 99 \\',
  '  -ngld 99 \\',
  '  --host 0.0.0.0 \\',
  '  --port 8091 \\',
  '  --ctx-size 16384 \\',
  '  -np 1 \\',
  '  -b 1024 \\',
  '  -ub 256 \\',
  '  --flash-attn on \\',
  '  --cache-type-k q8_0 \\',
  '  --cache-type-v q8_0 \\',
  '  --cache-type-k-draft q8_0 \\',
  '  --cache-type-v-draft q8_0 \\',
  '  --temp 0.7 \\',
  '  --top-p 0.95 \\',
  '  --top-k 20',
]
const typedTerminalLines = ref(terminalLines.map(() => ''))
const terminalFinished = ref(false)
const annotationVisible = ref(false)
const terminalPanel = ref(null)
const heroSection = ref(null)
const whySection = ref(null)
let terminalTimer = null
let annotationTimer = null
const heroCards = [
  {
    id: 'landing',
    src: '/assets/screenshots/newlanding.png',
    alt: 'llmctl TUI showing models, profiles, running services, and status panes',
  },
  {
    id: 'dashboard',
    src: '/assets/screenshots/dashboard.png',
    alt: 'llmctl dashboard showing active models, source trends, and GPU utilization',
  },
]

const angle = 12
const hoverResetDelay = 400
const hoverBuffer = 24
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

const clearHoverReset = () => {
  if (hoverResetTimer !== null) {
    window.clearTimeout(hoverResetTimer)
    hoverResetTimer = null
  }
}

const stopTerminalTyping = () => {
  if (terminalTimer !== null) {
    window.clearTimeout(terminalTimer)
    terminalTimer = null
  }
}

const stopAnnotationTimer = () => {
  if (annotationTimer !== null) {
    window.clearTimeout(annotationTimer)
    annotationTimer = null
  }
}

const startTerminalTyping = () => {
  stopTerminalTyping()
  stopAnnotationTimer()
  annotationVisible.value = false
  const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  if (reducedMotion) {
    typedTerminalLines.value = [...terminalLines]
    terminalFinished.value = true
    annotationVisible.value = true
    return
  }

  typedTerminalLines.value = terminalLines.map(() => '')
  terminalFinished.value = false
  annotationTimer = window.setTimeout(() => {
    annotationVisible.value = true
    annotationTimer = null
  }, 1500)

  const typeLine = (lineIndex, charIndex) => {
    if (lineIndex >= terminalLines.length) {
      terminalFinished.value = true
      terminalTimer = null
      return
    }

    const line = terminalLines[lineIndex]
    typedTerminalLines.value[lineIndex] = line.slice(0, charIndex)

    if (charIndex < line.length) {
      terminalTimer = window.setTimeout(() => typeLine(lineIndex, charIndex + 1), 12)
      return
    }

    terminalTimer = window.setTimeout(() => typeLine(lineIndex + 1, 0), 180)
  }

  terminalTimer = window.setTimeout(() => typeLine(0, 1), 220)
}

const queueHoverReset = (heroId, cardEl) => {
  clearHoverReset()

  hoverResetTimer = window.setTimeout(() => {
    hoverResetTimer = null
    heroTrackingActive = false

    if (activeHero.value !== heroId) {
      return
    }

    if (stopIntroMotion) {
      stopIntroMotion()
      stopIntroMotion = null
    }

    resetCardPose(cardEl)
    stopIntroMotion = startIntroMotion()
  }, hoverResetDelay)
}

const handleCardEnter = () => {
  clearHoverReset()
  heroTrackingActive = true

  if (stopIntroMotion) {
    stopIntroMotion()
    stopIntroMotion = null
  }
}

const handleCardMove = (event) => {
  if (stopIntroMotion) {
    stopIntroMotion()
    stopIntroMotion = null
  }

  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    return
  }

  const stackEl = event.currentTarget
  const cardEl = stackEl?.querySelector('.hero-stack-card.is-front')
  const imageEl = cardEl?.querySelector('.hero-card-image')
  if (!cardEl || !imageEl) {
    return
  }

  const rect = imageEl.getBoundingClientRect()
  const bufferRect = {
    left: rect.left - hoverBuffer,
    right: rect.right + hoverBuffer,
    top: rect.top - hoverBuffer,
    bottom: rect.bottom + hoverBuffer,
  }

  const insideImage =
    event.clientX >= rect.left &&
    event.clientX <= rect.right &&
    event.clientY >= rect.top &&
    event.clientY <= rect.bottom

  const insideBuffer =
    event.clientX >= bufferRect.left &&
    event.clientX <= bufferRect.right &&
    event.clientY >= bufferRect.top &&
    event.clientY <= bufferRect.bottom

  if (insideImage) {
    heroTrackingActive = true
    clearHoverReset()
  } else if (insideBuffer) {
    if (!heroTrackingActive) {
      return
    }
    clearHoverReset()
  } else {
    if (heroTrackingActive && hoverResetTimer === null) {
      queueHoverReset(activeHero.value, cardEl)
    }
    return
  }

  const x = event.clientX - (rect.left + rect.width / 2)
  const y = event.clientY - (rect.top + rect.height / 2)
  setCardPose(cardEl, remap(x, rect.width / 2, angle), remap(y, rect.height / 2, angle) * -1)
}

const handleCardLeave = (event) => {
  const currentEl = event.currentTarget
  const cardEl = currentEl?.classList?.contains('hero-stack-card')
    ? currentEl
    : currentEl?.closest?.('.hero-stack-card') ?? currentEl?.querySelector?.('.hero-stack-card.is-front') ?? null
  if (!cardEl) {
    return
  }

  if (!cardEl.classList.contains('is-front')) {
    return
  }

  queueHoverReset(activeHero.value, cardEl)
}

let hoverResetTimer = null
let heroTrackingActive = false

const getFrontCardImage = () => document.querySelector('.hero-stack-card.is-front .hero-card-image')

const startIntroMotion = () => {
  const image = getFrontCardImage()
  if (!image || window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    return () => {}
  }

  const card = image.closest('.hero-stack-card')
  if (!card) {
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

    setCardPose(card, rotateX, rotateY)

    if (progress < 1) {
      frameId = window.requestAnimationFrame(tick)
      return
    }

    resetCardPose(card)
  }

  frameId = window.requestAnimationFrame(tick)

  const stop = () => {
    stopped = true
    if (frameId !== null) {
      window.cancelAnimationFrame(frameId)
    }
    resetCardPose(card)
  }

  const stopOnUser = () => stop()
  image.addEventListener('pointermove', stopOnUser, { once: true, passive: true })
  image.addEventListener('pointerdown', stopOnUser, { once: true })
  image.addEventListener('click', stopOnUser, { once: true })

  return stop
}

let stopIntroMotion = null
let cleanupHero = null
let revealObserver = null
let terminalObserver = null
let terminalTypingStarted = false
let terminalTypingArmed = true
let previousScrollSnapType = ''
let previousScrollPaddingTop = ''
let previousScrollBehavior = ''
let previousBodyScrollSnapType = ''
let previousBodyScrollPaddingTop = ''
let previousBodyScrollBehavior = ''
let snapEnabled = false
let snapWheelHandler = null

const startTerminalTypingOnce = () => {
  if (terminalTypingStarted || !terminalTypingArmed) {
    return
  }

  terminalTypingStarted = true
  startTerminalTyping()
}

const resetWhySectionState = () => {
  stopTerminalTyping()
  stopAnnotationTimer()
  annotationVisible.value = false
  terminalFinished.value = false
  typedTerminalLines.value = terminalLines.map(() => '')
  terminalTypingStarted = false
  terminalTypingArmed = false

  if (whySection.value && revealObserver) {
    whySection.value.querySelectorAll('.reveal.visible').forEach((el) => {
      el.classList.remove('visible')
    })
  }
}

const armWhySectionReplay = () => {
  terminalTypingArmed = true

  if (whySection.value && revealObserver) {
    whySection.value.querySelectorAll('.reveal').forEach((el) => {
      revealObserver.observe(el)
    })
  }

  if (terminalPanel.value && terminalObserver) {
    terminalObserver.observe(terminalPanel.value)
  }
}

onMounted(() => {
  snapEnabled = window.matchMedia('(min-width: 961px)').matches
  if (snapEnabled) {
    const scrollRoot = document.scrollingElement || document.documentElement
    previousScrollSnapType = document.documentElement.style.scrollSnapType
    previousScrollPaddingTop = document.documentElement.style.scrollPaddingTop
    previousScrollBehavior = document.documentElement.style.scrollBehavior
    previousBodyScrollSnapType = document.body.style.scrollSnapType
    previousBodyScrollPaddingTop = document.body.style.scrollPaddingTop
    previousBodyScrollBehavior = document.body.style.scrollBehavior
    scrollRoot.style.scrollSnapType = 'y proximity'
    scrollRoot.style.scrollPaddingTop = 'calc(var(--vp-nav-height) + 1rem)'
    scrollRoot.style.scrollBehavior = 'smooth'
    document.documentElement.style.scrollSnapType = 'y proximity'
    document.documentElement.style.scrollPaddingTop = 'calc(var(--vp-nav-height) + 1rem)'
    document.documentElement.style.scrollBehavior = 'smooth'
    document.body.style.scrollSnapType = 'y proximity'
    document.body.style.scrollPaddingTop = 'calc(var(--vp-nav-height) + 1rem)'
    document.body.style.scrollBehavior = 'smooth'
  }

  revealObserver = new IntersectionObserver((entries) => {
    entries.forEach(e => {
      if (e.isIntersecting) {
        e.target.classList.add('visible')
        revealObserver?.unobserve(e.target)
      }
    })
  }, { threshold: .15, rootMargin: '0px 0px -40px 0px' })

  document.querySelectorAll('.reveal').forEach(el => revealObserver?.observe(el))

  terminalObserver = new IntersectionObserver((entries) => {
    entries.forEach((entry) => {
      if (!entry.isIntersecting) {
        return
      }

      startTerminalTypingOnce()
      terminalObserver?.unobserve(entry.target)
    })
  }, { threshold: .25, rootMargin: '0px 0px -10% 0px' })

  if (terminalPanel.value) {
    terminalObserver.observe(terminalPanel.value)
  }

  if (snapEnabled) {
    snapWheelHandler = (event) => {
      if (!heroSection.value || !marqueeWrap.value) {
        return
      }

      const scroller = document.scrollingElement || document.documentElement
      const navHeight = parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--vp-nav-height')) || 0
      const snapGap = 0
      const downTargetTop = marqueeWrap.value.offsetTop - navHeight - snapGap
      const upTargetTop = heroSection.value.offsetTop - navHeight - 24
      const canSnapDown = event.deltaY > 0 && window.scrollY < downTargetTop - 48
      const canSnapUp = event.deltaY < 0 && window.scrollY > downTargetTop - 170

      if (!canSnapDown && !canSnapUp) {
        return
      }

      event.preventDefault()

      if (canSnapDown) {
        armWhySectionReplay()
        scroller.scrollTo({ top: downTargetTop, behavior: 'smooth' })
        return
      }

      resetWhySectionState()
      scroller.scrollTo({ top: upTargetTop, behavior: 'smooth' })
    }

    window.addEventListener('wheel', snapWheelHandler, { passive: false })
  }

  stopIntroMotion = startIntroMotion()
  cleanupHero = () => {
    clearHoverReset()
    stopTerminalTyping()
    stopAnnotationTimer()
    revealObserver?.disconnect()
    revealObserver = null
    terminalObserver?.disconnect()
    terminalObserver = null
    if (stopIntroMotion) {
      stopIntroMotion()
      stopIntroMotion = null
    }
    terminalTypingStarted = false
    terminalTypingArmed = true
    if (snapEnabled) {
      document.documentElement.style.scrollSnapType = previousScrollSnapType
      document.documentElement.style.scrollPaddingTop = previousScrollPaddingTop
      document.documentElement.style.scrollBehavior = previousScrollBehavior
      document.body.style.scrollSnapType = previousBodyScrollSnapType
      document.body.style.scrollPaddingTop = previousBodyScrollPaddingTop
      document.body.style.scrollBehavior = previousBodyScrollBehavior
    }
    if (snapWheelHandler) {
      window.removeEventListener('wheel', snapWheelHandler)
      snapWheelHandler = null
    }
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
    <section ref="heroSection" class="hero">
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
        :style="{ '--hero-url': `url(${to('/assets/screenshots/newlanding.png')})` }"
        @pointermove="handleCardMove"
        @pointerleave="handleCardLeave"
        @pointercancel="handleCardLeave"
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
          <div class="hero-card-frame">
            <div class="hero-card-glow"></div>
            <img
              :src="to(card.src)"
              :alt="card.alt"
              loading="eager"
              class="hero-card-image"
              @pointerenter="handleCardEnter"
              @click="toggleHeroFront(card.id)"
              @pointerout="handleCardLeave"
            />
            <div class="hero-card-sheen" aria-hidden="true"></div>
          </div>
        </button>
      </div>

      <div class="hero-scroll">
        <span>Scroll</span>
        <div class="scroll-line"></div>
      </div>
    </section>

    <section ref="marqueeWrap" class="marquee-wrap" aria-label="Highlights">
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
    </section>

    <section ref="whySection" class="why-section">
      <div class="why-annotation" :class="{ visible: annotationVisible }" aria-hidden="true">
        <span class="why-annotation-text">
          <span>This is</span>
          <span>why</span>
        </span>
        <svg class="why-annotation-arrow" viewBox="0 0 920 340" preserveAspectRatio="none">
          <path class="why-annotation-line" d="M 216 186 C 250 186, 282 190, 312 198 C 348 208, 386 214, 422 220 C 456 226, 476 232, 494 240" />
          <path class="why-annotation-line why-annotation-head" d="M 494 240 C 482 234, 473 227, 464 218" />
          <path class="why-annotation-line why-annotation-head" d="M 494 240 C 482 246, 473 254, 465 264" />
        </svg>
      </div>
      <div class="why-copy">
        <div class="section-label reveal">Why llmctl?</div>
        <h2 class="section-title reveal reveal-delay-1">
          Why?<br>
          <span style="color:var(--text2)">Flag configurations and spot-benchmarking is painful.</span>
        </h2>
        <p class="section-subtitle reveal reveal-delay-2">
          llmctl keeps your model launch workflows, profile presets, and RPC offload setups in one place so you can move from idea to run without rebuilding the same command line over and over.
        </p>

        <div class="why-links reveal reveal-delay-3">
          <a class="why-link" :href="to('/quickstart')">
            <span>Quickstart</span>
            <small>Get to a first run fast.</small>
          </a>
          <a class="why-link" :href="to('/guides/profiles')">
            <span>Profiles</span>
            <small>Save launch flags once, reuse them later.</small>
          </a>
          <a class="why-link" :href="to('/guides/rpc')">
            <span>RPC</span>
            <small>Split layers across machines when you need it.</small>
          </a>
          <a class="why-link" :href="to('/reference/tui')">
            <span>TUI reference</span>
            <small>See the full terminal workflow at a glance.</small>
          </a>
        </div>
      </div>

      <div ref="terminalPanel" class="terminal-panel reveal reveal-delay-1" aria-label="Example llama-server command">
        <div class="terminal-topbar">
          <span class="terminal-dot red"></span>
          <span class="terminal-dot yellow"></span>
          <span class="terminal-dot green"></span>
          <span class="terminal-title">bash</span>
        </div>
        <div class="terminal-body">
          <div
            v-for="(line, index) in typedTerminalLines"
            :key="`${index}-${line}`"
            class="terminal-line"
          >
            <span v-if="index === 0" class="terminal-prompt">❯ </span>
            <span class="terminal-text">{{ line }}</span>
            <span v-if="terminalFinished && index === typedTerminalLines.length - 1" class="terminal-cursor">█</span>
          </div>
          <div v-if="!terminalFinished" class="terminal-line terminal-line-pulse">
            <span class="terminal-prompt">❯ </span>
            <span class="terminal-text terminal-placeholder">typing command...</span>
            <span class="terminal-cursor">█</span>
          </div>
        </div>
      </div>
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
  padding: .1rem .1rem .1rem;
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
  position: relative;
  scroll-snap-align: start;
  scroll-snap-stop: always;
}


.hero-copy  { grid-column: 1; grid-row: 1; }
.hero-media {
  grid-column: 2;
  grid-row: 1;
  justify-self: end;
  width: min(100%, 720px);
}

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
.primary-btn:focus-visible,
.secondary-btn:hover,
.secondary-btn:focus-visible {
  transform: translateY(-2px);
}

.primary-btn:hover,
.primary-btn:focus-visible {
  box-shadow: 0 12px 32px rgba(0, 229, 255, 0.3);
}

.secondary-btn:hover,
.secondary-btn:focus-visible {
  border-color: rgba(0, 229, 255, 0.25);
  background: rgba(255, 255, 255, 0.07);
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
  opacity: 0.01;
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

.why-section {
  display: grid;
  grid-template-columns: minmax(0, 1.05fr) minmax(320px, 0.95fr);
  gap: 1.5rem;
  align-items: stretch;
  padding: 3rem 0 1.25rem;
  position: relative;
  scroll-snap-align: start;
  scroll-snap-stop: always;
  scroll-margin-top: calc(var(--vp-nav-height));
}

.why-annotation {
  position: absolute;
  inset: -1.2rem 0 auto 0;
  height: 272px;
  pointer-events: none;
  opacity: 0;
  transform: translateY(8px) scale(0.98);
  transition: opacity .4s ease, transform .4s ease;
  z-index: 2;
}

.why-annotation.visible {
  opacity: 1;
  transform: translateY(0) scale(1);
}

.why-annotation-text {
  position: absolute;
  left: 22%;
  top: 3rem;
  display: flex;
  flex-direction: column;
  gap: .05rem;
  font-family: "Segoe Print", "Bradley Hand", "Comic Sans MS", cursive;
  font-size: clamp(1.35rem, 1.8vw, 2.05rem);
  line-height: .84;
  color: #ff3b30;
  letter-spacing: -.04em;
  transform: rotate(-6deg);
  text-shadow: 0 0 0.5px rgba(255, 59, 48, 0.5);
}

.why-annotation-text span:last-child {
  margin-left: 1.65rem;
}

.why-annotation-arrow {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  overflow: visible;
}

.why-annotation-line {
  fill: none;
  stroke: #ff3b30;
  stroke-width: 4.75;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-dasharray: 710;
  stroke-dashoffset: 710;
  filter: drop-shadow(0 0 1px rgba(255, 59, 48, 0.24));
}

.why-annotation.visible .why-annotation-line {
  animation: drawArrow 1.35s ease forwards;
}

.why-annotation-head {
  animation-delay: .9s;
}

.why-copy {
  padding: 1rem 0;
}

.why-links {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: .85rem;
  margin-top: 2rem;
}

.why-link {
  display: flex;
  flex-direction: column;
  gap: .3rem;
  padding: 1rem 1.1rem;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.03);
  text-decoration: none;
  color: inherit;
  transition: transform .2s ease, border-color .2s ease, background .2s ease, box-shadow .2s ease;
}

.why-link:hover,
.why-link:focus-visible {
  transform: translateY(-2px);
  border-color: rgba(0, 229, 255, 0.25);
  background: rgba(255, 255, 255, 0.06);
  box-shadow: 0 12px 28px rgba(0, 0, 0, 0.22);
  outline: none;
}

.why-link span {
  font-size: .85rem;
  font-weight: 700;
  letter-spacing: .08em;
  text-transform: uppercase;
  color: #f2f6ff;
}

.why-link small {
  font-size: .9rem;
  line-height: 1.5;
  color: rgba(232, 232, 240, 0.72);
}

.terminal-panel {
  display: flex;
  flex-direction: column;
  min-height: 100%;
  border-radius: 22px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background:
    linear-gradient(180deg, rgba(12, 16, 26, 0.96), rgba(8, 10, 18, 0.94)),
    rgba(255, 255, 255, 0.03);
  box-shadow:
    0 24px 60px rgba(0, 0, 0, 0.32),
    0 0 0 1px rgba(255, 255, 255, 0.03) inset;
  overflow: hidden;
}

.terminal-topbar {
  display: flex;
  align-items: center;
  gap: .45rem;
  padding: .9rem 1rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(255, 255, 255, 0.03);
}

.terminal-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  display: inline-block;
}

.terminal-dot.red { background: #ff5f57; }
.terminal-dot.yellow { background: #febc2e; }
.terminal-dot.green { background: #28c840; }

.terminal-title {
  margin-left: .45rem;
  font-size: .75rem;
  letter-spacing: .12em;
  text-transform: uppercase;
  color: rgba(232, 232, 240, 0.55);
}

.terminal-body {
  padding: 1.2rem 1.1rem 1.35rem;
  font-family: 'JetBrains Mono', 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
  font-size: .84rem;
  line-height: 1.85;
  color: #d9e7ff;
}

.terminal-line {
  display: flex;
  align-items: flex-start;
  white-space: pre-wrap;
  word-break: break-word;
}

.terminal-prompt {
  color: var(--accent);
  margin-right: .55rem;
}

.terminal-text {
  color: #d9e7ff;
}

.terminal-placeholder {
  color: rgba(232, 232, 240, 0.42);
}

.terminal-cursor {
  display: inline-block;
  margin-left: .2rem;
  color: var(--accent);
  animation: cursorBlink 1s steps(1) infinite;
}

.terminal-line-pulse .terminal-cursor {
  opacity: 1;
}

.terminal-line:not(:first-child) .terminal-prompt {
  opacity: 0;
  width: 0;
}

.marquee-wrap {
  padding: 1.1rem 0;
  overflow: hidden;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  scroll-margin-top: calc(var(--vp-nav-height) + .25rem);
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

  .why-section {
    grid-template-columns: 1fr;
    gap: 1rem;
    padding: 3rem 0 1rem;
  }

  .why-copy {
    padding: 0;
  }

  .why-links {
    grid-template-columns: 1fr;
  }

  .why-annotation {
    display: none;
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

  .terminal-body {
    font-size: .74rem;
    line-height: 1.45;
  }

  .terminal-panel {
    min-height: 0;
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

@keyframes cursorBlink {
  0%, 49% { opacity: 1; }
  50%, 100% { opacity: 0; }
}

@keyframes drawArrow {
  to {
    stroke-dashoffset: 0;
  }
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
