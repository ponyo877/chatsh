import { defineConfig } from 'vite'

// https://vitejs.dev/config/
export default defineConfig({
    // No longer need viteCommonjs plugin as @bufbuild/protoc-gen-es generates ES modules
    // plugins: [], 
    server: {
        port: 3000,
        open: true
    },
    build: {
        outDir: 'dist'
    },
    define: {
        // This might still be useful for some libraries, can be kept.
        global: 'globalThis',
    },
    optimizeDeps: {
        // Include Connect and Protobuf runtimes
        include: ['@bufbuild/connect', '@bufbuild/connect-web', '@bufbuild/protobuf']
    }
})
