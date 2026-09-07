import { defineConfig } from "vite";

export default defineConfig({
  build: {
    emptyOutDir: true,
    outDir: "dist/preload",
    ssr: "src/preload.ts",
    rollupOptions: {
      external: ["electron"],
      output: {
        entryFileNames: "preload.cjs",
        format: "cjs",
      },
    },
  },
});
