/* eslint-disable react-refresh/only-export-components */
import { StaticRouter } from "react-router";
import { prerender } from "react-dom/static";
import { AppProviders, AppShell } from "./App";

export { PUBLIC_PAGE_META } from "./seo-meta";

/**
 * 构建期预渲染入口（scripts/prerender.mjs 调用）。
 * 用 react-dom/static 的 prerender 而非 renderToString：
 * 公开页多为 lazy() 组件，prerender 会等待所有 Suspense 边界完成。
 */
export async function render(url: string): Promise<string> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 30_000);
  try {
    const { prelude } = await prerender(
      <AppProviders>
        <StaticRouter location={url}>
          <AppShell />
        </StaticRouter>
      </AppProviders>,
      { signal: controller.signal },
    );
    const reader = prelude.getReader();
    const decoder = new TextDecoder();
    let html = "";
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      html += decoder.decode(value, { stream: true });
    }
    return html;
  } finally {
    clearTimeout(timer);
  }
}
