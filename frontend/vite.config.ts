import tailwindcss from "@tailwindcss/vite";
import { devtools } from "@tanstack/devtools-vite";

import { tanstackStart } from "@tanstack/react-start/plugin/vite";

import viteReact from "@vitejs/plugin-react";
import { nitro } from "nitro/vite";
import { defineConfig, loadEnv } from "vite";

const config = defineConfig(({ mode }) => {
	const env = loadEnv(mode, process.cwd(), "");
	return {
		server: {
			port: Number(process.env.PORT || 3000),
			watch: {
				usePolling: env.DEV_POLLING === "true",
				interval: 500,
				ignored: ["**/.output/**", "**/.nitro/**"],
			},
			proxy: env.API_PROXY_TARGET
				? {
						"/api": { target: env.API_PROXY_TARGET, changeOrigin: false },
					}
				: undefined,
		},
		resolve: { tsconfigPaths: true },
		plugins: [
			devtools(),
			nitro({ rollupConfig: { external: [/^@sentry\//] } }),
			tailwindcss(),
			tanstackStart(),
			viteReact(),
		],
	};
});

export default config;
