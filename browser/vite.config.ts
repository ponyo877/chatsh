import { defineConfig } from 'vite'

// https://vitejs.dev/config/
export default defineConfig({
    // plugins: [], // No plugins needed for this basic setup yet
    server: {
        port: 3000, // Optional: specify dev server port
        open: true    // Optional: open browser on server start
    },
    build: {
        outDir: 'dist' // Optional: specify output directory
    }
})
