import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
    plugins: [react(), tailwindcss()],
    base: './',


    clearScreen: false,

    server: {
        port: 1420,
        strictPort: true,
        proxy: {
            '/api': {
                target: 'http://127.0.0.1:9201',
                changeOrigin: false,
                secure: false,
            },
        },
    },
    envPrefix: ['VITE_', 'TAURI_'],
    build: {
        // Tauri v2 uses better defaults but let's be explicit for modern web features
        target: 'esnext',
        minify: !process.env.TAURI_DEBUG ? 'esbuild' : false,
        sourcemap: !!process.env.TAURI_DEBUG,
    },
});
