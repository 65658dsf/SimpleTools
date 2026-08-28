import { writeFileSync } from 'node:fs'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [
    vue(),
    {
      name: 'keep-wails-embed-directory',
      closeBundle() {
        // Go's //go:embed requires this directory to exist for backend-only
        // builds, while Vite clears dist before writing production assets.
        writeFileSync('dist/.gitkeep', 'Wails embed directory placeholder.\n')
      },
    },
  ],
  server: { port: 4173, strictPort: false },
})
