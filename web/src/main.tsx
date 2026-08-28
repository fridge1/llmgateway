import { createRoot, hydrateRoot } from "react-dom/client";
import App from "./App.tsx";
import "./index.css";

const container = document.getElementById("root")!;
// 预渲染的公开页（scripts/prerender.mjs 产物）带有服务端 HTML → hydrate 复用；
// 纯 SPA 壳（登录后路由等）→ 常规挂载。
if (container.hasChildNodes()) {
  hydrateRoot(container, <App />);
} else {
  createRoot(container).render(<App />);
}
