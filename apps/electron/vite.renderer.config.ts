import { fileURLToPath, URL } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ command }) => ({
  base: "./",
  // Concurrent desktop/preview servers must not overwrite each other's optimized
  // dependencies: that leaves live editors importing expired module hashes (504).
  cacheDir: fileURLToPath(new URL(
    command === "serve"
      ? `./node_modules/.vite/aivo-renderer-${process.pid}`
      : "./node_modules/.vite/aivo-renderer-build",
    import.meta.url,
  )),
  plugins: [
    tanstackRouter({
      autoCodeSplitting: true,
      generatedRouteTree: "routeTree.gen.ts",
      routesDirectory: "routes",
    }),
    react({}),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  root: "src",
  build: {
    emptyOutDir: true,
    outDir: "../dist/renderer",
  },
}));
