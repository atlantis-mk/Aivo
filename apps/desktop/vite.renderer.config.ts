import { fileURLToPath, URL } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  base: "./",
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
});
