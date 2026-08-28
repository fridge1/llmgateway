/**
 * 构建期预渲染脚本：把公开页渲染为静态 HTML，让百度等不执行 JS 的
 * 爬虫拿到正文。产物：
 *   dist/__prerendered/index.html  ← 首页（nginx `location = /` 指向；
 *                                     不直接覆盖 dist/index.html，否则
 *                                     /dashboard 等深链接首屏会闪现首页内容）
 *   dist/claude/index.html 等      ← 子路由（nginx try_files $uri/ 自动命中）
 *
 * 设计为永不阻塞构建：单路由渲染失败仅告警并跳过，该路由回退 SPA 壳。
 */
import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const dist = path.join(root, "dist");
const template = readFileSync(path.join(dist, "index.html"), "utf-8");

const { render, PUBLIC_PAGE_META } = await import(
  pathToFileURL(path.join(root, "dist-ssr", "entry-server.js")).href
);

const SITE_URL = "https://your-domain.com";

const escapeHtml = (s) =>
  s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");

/** 用页面 TDK 重写模板 head，并补 canonical */
function applyHead(html, meta, route) {
  const url = `${SITE_URL}${route === "/" ? "/" : route}`;
  return html
    .replace(/<title>[^<]*<\/title>/, `<title>${escapeHtml(meta.title)}</title>`)
    .replace(/(<meta name="description" content=")[^"]*(")/, `$1${escapeHtml(meta.description)}$2`)
    .replace(/(<meta property="og:title" content=")[^"]*(")/, `$1${escapeHtml(meta.title)}$2`)
    .replace(/(<meta property="og:description" content=")[^"]*(")/, `$1${escapeHtml(meta.description)}$2`)
    .replace(/(<meta property="og:url" content=")[^"]*(")/, `$1${url}$2`)
    .replace("</head>", `  <link rel="canonical" href="${url}" />\n  </head>`);
}

/**
 * React 19 会把 Seo 组件的 <title>/<meta>/<link> 按 hoistable 规则输出，
 * 在非完整文档渲染下这些标签会混在 body 流里 —— head 已由 applyHead 处理，
 * 把 body 里的这份剥掉，避免正文中出现重复的 meta。
 */
function stripHoistedMeta(appHtml) {
  return appHtml
    .replace(/<title>[^<]*<\/title>/g, "")
    .replace(/<meta (?:name="(?:description|robots)"|property="og:[^"]*") content="[^"]*"\/?>/g, "")
    .replace(/<link rel="canonical" href="[^"]*"\/?>/g, "");
}

let ok = 0;
let failed = 0;
for (const [route, meta] of Object.entries(PUBLIC_PAGE_META)) {
  try {
    const appHtml = stripHoistedMeta(await render(route));
    if (appHtml.trim().length < 500) {
      throw new Error(`rendered output suspiciously small (${appHtml.length} chars)`);
    }
    let html = applyHead(template, meta, route);
    html = html.replace('<div id="root"></div>', `<div id="root">${appHtml}</div>`);
    const outFile =
      route === "/"
        ? path.join(dist, "__prerendered", "index.html")
        : path.join(dist, route.replace(/^\//, ""), "index.html");
    mkdirSync(path.dirname(outFile), { recursive: true });
    writeFileSync(outFile, html);
    ok++;
    console.log(`[prerender] ok: ${route} -> ${path.relative(dist, outFile)} (${html.length} bytes)`);
  } catch (err) {
    failed++;
    console.warn(`[prerender] FAILED（该路由回退 SPA 壳，不阻塞构建）: ${route}`);
    console.warn(err);
  }
}
console.log(`[prerender] done: ${ok} ok, ${failed} failed`);
