import { defineConfig, mergeConfig } from "vite";

import appConfig from "../vite.config";

export default mergeConfig(
  appConfig,
  defineConfig({
    server: {
      host: "127.0.0.1",
      port: 4173,
      strictPort: true,
      origin: "http://127.0.0.1:4173",
      hmr: {
        protocol: "ws",
        host: "127.0.0.1",
        port: 4173,
        clientPort: 4173,
      },
    },
  }),
);
