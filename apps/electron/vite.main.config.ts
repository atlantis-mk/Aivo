import { defineConfig } from "vite";

export default defineConfig({
  build: {
    emptyOutDir: true,
    outDir: "dist/main",
    ssr: "src/main.ts",
    rollupOptions: {
      external: ["electron"],
      output: {
        entryFileNames: "main.cjs",
        format: "cjs",
      },
    },
  },
});
