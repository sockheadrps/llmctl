import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/llmctl/',
  title: 'llmctl',
  description: 'A TUI-first tool for running, configuring, and distributing local llama.cpp models.',
  themeConfig: {
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Quickstart', link: '/quickstart' },
      { text: 'Guides', link: '/guides/local-models' },
      { text: 'Reference', link: '/reference/tui' }
    ],
    sidebar: {
      '/': [
        {
          text: 'Start Here',
          items: [
            { text: 'Overview', link: '/' },
            { text: 'Installation', link: '/installation' },
            { text: 'Quickstart', link: '/quickstart' },
            { text: 'Concepts', link: '/concepts' }
          ]
        },
        {
          text: 'Guides',
          items: [
            { text: 'Local Models', link: '/guides/local-models' },
            { text: 'Profiles', link: '/guides/profiles' },
            { text: 'RPC', link: '/guides/rpc' },
            { text: 'Templates', link: '/guides/templates' },
            { text: 'Benchmarking', link: '/guides/benchmarking' },
            { text: 'Status Server', link: '/guides/status-server' },
            { text: 'Troubleshooting', link: '/guides/troubleshooting' }
          ]
        },
        {
          text: 'Reference',
          items: [
            { text: 'TUI', link: '/reference/tui' },
            { text: 'Profile Options', link: '/reference/profile-options' },
            { text: 'Config Schema', link: '/reference/config-schema' },
            { text: 'CLI', link: '/reference/cli' }
          ]
        }
      ]
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/sockheadrps/llmctl' }
    ]
  }
})
