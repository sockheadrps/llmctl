import DefaultTheme from 'vitepress/theme'
import LandingPage from './components/LandingPage.vue'
import './custom.css'

let scrollListenerInstalled = false

export default {
  ...DefaultTheme,
  enhanceApp(ctx) {
    DefaultTheme.enhanceApp?.(ctx)
    ctx.app.component('LandingPage', LandingPage)

    if (typeof window !== 'undefined' && !scrollListenerInstalled) {
      scrollListenerInstalled = true
      const root = document.documentElement

      const update = () => {
        root.classList.toggle('docs-scrolled', window.scrollY > 18)
      }

      update()
      window.addEventListener('scroll', update, { passive: true })
      window.addEventListener('resize', update, { passive: true })
    }
  }
}
