import { PUBLIC_PAGE_META, SITE_URL } from "@/seo-meta";

interface SeoProps {
  /** 路由路径（如 "/claude"），用于 canonical/og:url；公开页会自动查 PUBLIC_PAGE_META */
  path: string;
  /** 不在 PUBLIC_PAGE_META 中的页面（如登录页）显式传入 */
  title?: string;
  description?: string;
  /** 登录页等不希望被收录的页面置 true */
  noindex?: boolean;
}

/**
 * 轻量 SEO 组件：React 19 会把组件内渲染的 <title>/<meta>/<link>
 * 自动提升（hoist）到 <head>，无需 react-helmet。
 *
 * 注意：index.html 中的静态 title/description 作为未挂载本组件页面
 * （登录后控制台等）的兜底；公开页的静态标签由预渲染脚本
 * （scripts/prerender.mjs）在构建期替换，消除双份问题。
 */
export function Seo({ path, title, description, noindex = false }: SeoProps) {
  const meta = PUBLIC_PAGE_META[path];
  const fullTitle = title ?? meta?.title ?? "LLM Gateway";
  const desc = description ?? meta?.description ?? "";
  const url = `${SITE_URL}${path === "/" ? "/" : path}`;
  return (
    <>
      <title>{fullTitle}</title>
      <meta name="description" content={desc} />
      <link rel="canonical" href={url} />
      <meta property="og:title" content={fullTitle} />
      <meta property="og:description" content={desc} />
      <meta property="og:url" content={url} />
      {noindex && <meta name="robots" content="noindex, nofollow" />}
    </>
  );
}

/** 渲染一段 JSON-LD 结构化数据（必应/Google 富结果；百度不消费但无害） */
export function JsonLd({ data }: { data: Record<string, unknown> }) {
  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data) }}
    />
  );
}
