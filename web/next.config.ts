import type { NextConfig } from "next";
import path from "node:path";

const nextConfig: NextConfig = {
  // Silencia la detección ambigua de workspace root: hay un package-lock.json
  // residual en la raíz del repo (proyecto Node/TS abandonado al pivotar el
  // backend a Go), separado de este proyecto Next.js.
  turbopack: {
    root: path.join(__dirname),
  },
};

export default nextConfig;
