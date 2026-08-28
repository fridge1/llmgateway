import { ArrowLeft, ShieldCheck, Zap } from "lucide-react";
import { Link } from "react-router-dom";
import BeianBar from "@/components/BeianBar";
import { Seo } from "@/components/Seo";
import { SITE } from "@/config/site";

const LICENSE_PAGES = [
  "/qualifications/telecom-license-page-1.webp",
  "/qualifications/telecom-license-page-2.webp",
];

const QualificationsPage = () => (
  <div className="min-h-screen bg-muted/30 text-foreground">
    <Seo path="/qualifications" />
    <header className="border-b border-border bg-background">
      <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-6">
        <Link to="/" className="flex items-center gap-2.5" aria-label="返回 LLM Gateway 首页">
          <span className="brand-gradient flex h-8 w-8 items-center justify-center rounded-lg shadow-button">
            <Zap size={15} className="text-white" />
          </span>
          <span className="text-sm font-bold">LLM Gateway</span>
        </Link>
        <Link
          to="/"
          className="inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft size={16} />
          返回首页
        </Link>
      </div>
    </header>

    <main className="mx-auto max-w-5xl px-4 py-10 sm:px-6 sm:py-14">
      <div className="mb-8 text-center">
        <ShieldCheck className="mx-auto mb-3 h-8 w-8 text-primary" aria-hidden="true" />
        <h1 className="text-2xl font-bold sm:text-3xl">企业资质</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          增值电信业务经营许可证：{SITE.telecomLicenseNumber}
        </p>
      </div>

      <div className="mx-auto max-w-3xl space-y-6">
        {LICENSE_PAGES.map((src, index) => (
          <figure key={src} className="overflow-hidden rounded-lg border border-border bg-white shadow-sm">
            <img
              src={src}
              alt={`增值电信业务经营许可证第 ${index + 1} 页`}
              className="block h-auto w-full"
              loading={index === 0 ? "eager" : "lazy"}
            />
          </figure>
        ))}
      </div>
    </main>

    <footer className="border-t border-border bg-background px-6 py-5 text-center">
      <BeianBar />
    </footer>
  </div>
);

export default QualificationsPage;
