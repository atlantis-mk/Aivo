import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  base: "./",
  plugins: [react({})],
  root: "src",
  build: {
    emptyOutDir: true,
    outDir: "../dist/renderer",
  },
});
