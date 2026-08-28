import { useNavigate } from "react-router-dom";
import { useMemo, useState } from "react";
import { Download, Monitor, Cpu, ArrowRight, Zap, Check, Apple, ShieldAlert, Terminal, Copy } from "lucide-react";
import BeianBar from "@/components/BeianBar";
import { Seo } from "@/components/Seo";

const VERSION = "0.1.0";

type Platform = "macos" | "windows";

const DOWNLOADS: Record<Platform, Record<string, {
  label: string;
  chip: string;
  file: string;
  size: string;
  url: string;
}>> = {
  macos: {
    arm64: {
      label: "Apple Silicon",
      chip: "M1 / M2 / M3 / M4",
      file: `LLM-Gateway-Desktop_${VERSION}_aarch64.dmg`,
      size: "6.3 MB",
      url: `https://your-tos-bucket.tos-cn-beijing.volces.com/downloads/LLM-Gateway-Desktop_${VERSION}_aarch64.dmg`,
    },
    x64: {
      label: "Intel",
      chip: "Intel Core i5 / i7 / i9",
      file: `LLM-Gateway-Desktop_${VERSION}_x64.dmg`,
      size: "6.6 MB",
      url: `https://your-tos-bucket.tos-cn-beijing.volces.com/downloads/LLM-Gateway-Desktop_${VERSION}_x64.dmg`,
    },
  },
  windows: {
    x64: {
      label: "Windows x64",
      chip: "Intel / AMD 64 位处理器",
      file: `LLM-Gateway-Desktop_${VERSION}_x64-setup.exe`,
      size: "4.2 MB",
      url: `/tos-proxy/downloads/LLM-Gateway-Desktop_${VERSION}_x64-setup.exe`,
    },
    arm64: {
      label: "Windows ARM64",
      chip: "Snapdragon / 高通 ARM 处理器",
      file: `LLM-Gateway-Desktop_${VERSION}_arm64-setup.exe`,
      size: "3.8 MB",
      url: `/tos-proxy/downloads/LLM-Gateway-Desktop_${VERSION}_arm64-setup.exe`,
    },
  },
};

function detectOS(): Platform {
  const ua = navigator.userAgent.toLowerCase();
  if (ua.includes("win")) return "windows";
  return "macos";
}

function detectArch(): "arm64" | "x64" {
  const ua = navigator.userAgent.toLowerCase();
  if (ua.includes("arm64") || ua.includes("aarch64")) return "arm64";
  const platform =
    (navigator as Navigator & { userAgentData?: { platform?: string } })
      .userAgentData?.platform || navigator.platform || "";
  if (/arm|aarch64/i.test(platform)) return "arm64";
  return typeof navigator.hardwareConcurrency === "number" && navigator.hardwareConcurrency >= 8
    ? "arm64"
    : "x64";
}

const INSTALL_STEPS: Record<Platform, { step: string; title: string; desc: string }[]> = {
  macos: [
    { step: "1", title: "下载安装包", desc: "选择对应芯片版本的 DMG 文件" },
    { step: "2", title: "拖入应用程序", desc: "打开 DMG，将应用拖入 Applications 文件夹" },
    { step: "3", title: "解除安全限制", desc: "首次打开前需在终端执行一次命令（见下方）" },
    { step: "4", title: "登录并配置", desc: "打开应用，登录账号，一键配置 AI 工具" },
  ],
  windows: [
    { step: "1", title: "下载安装包", desc: "选择对应处理器版本的安装程序" },
    { step: "2", title: "运行安装程序", desc: "双击 .exe 文件，按提示完成安装" },
    { step: "3", title: "信任安全提示", desc: "首次运行时 SmartScreen 可能提示，点击「详细信息」→「仍要运行」" },
    { step: "4", title: "登录并配置", desc: "打开应用，登录账号，一键配置 AI 工具" },
  ],
};

const DownloadPage = () => {
  const navigate = useNavigate();
  const detectedOS = useMemo(() => detectOS(), []);
  const recommended = useMemo(() => detectArch(), []);
  const [platform, setPlatform] = useState<Platform>(detectedOS);
  const [copied, setCopied] = useState(false);

  const xattrCmd = "xattr -cr /Applications/LLM\\ Gateway\\ Desktop.app";
  const handleCopy = () => {
    navigator.clipboard.writeText(xattrCmd);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const downloads = DOWNLOADS[platform];
  const archKeys = Object.keys(downloads);

  return (
    <div className="w-full min-h-screen bg-background">
      <Seo path="/download" />
      {/* Hero */}
      <div className="relative overflow-hidden bg-gradient-to-br from-slate-900 via-indigo-900 to-slate-900">
        <div className="absolute inset-0 bg-grid-white/[0.05] bg-[size:60px_60px]" />
        <div className="absolute top-0 right-0 w-96 h-96 bg-indigo-500/30 rounded-full blur-3xl" />
        <div className="absolute bottom-0 left-0 w-96 h-96 bg-emerald-500/20 rounded-full blur-3xl" />

        <div className="relative max-w-5xl mx-auto px-6 py-20 text-center">
          <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-300 text-sm font-medium mb-6">
            <Monitor size={16} />
            Desktop 客户端
          </div>
          <h1 className="text-4xl md:text-5xl font-bold text-white mb-6 leading-tight">
            下载 LLM Gateway Desktop
          </h1>
          <p className="text-lg text-slate-300 mb-4 max-w-2xl mx-auto">
            一键配置 Claude Code、Codex CLI 等 AI 编码工具<br />
            自动管理 API Key 和模型设置，开箱即用
          </p>
          <p className="text-sm text-slate-400">
            当前版本 v{VERSION} · 支持 macOS / Windows
          </p>
        </div>
      </div>

      {/* Platform Toggle */}
      <div className="max-w-4xl mx-auto px-6 -mt-6 relative z-20 flex justify-center">
        <div className="inline-flex bg-card border border-border rounded-xl p-1 shadow-lg">
          {([
            { key: "macos" as Platform, label: "macOS", icon: <Apple size={16} /> },
            { key: "windows" as Platform, label: "Windows", icon: <Monitor size={16} /> },
          ]).map((p) => (
            <button
              key={p.key}
              onClick={() => setPlatform(p.key)}
              className={`flex items-center gap-2 px-5 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 ${
                platform === p.key
                  ? "bg-indigo-500 text-white shadow-md"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {p.icon}
              {p.label}
            </button>
          ))}
        </div>
      </div>

      {/* Download cards */}
      <div className="max-w-4xl mx-auto px-6 mt-6 relative z-10">
        <div className={`grid gap-6 ${archKeys.length > 1 ? "md:grid-cols-2" : "max-w-md mx-auto"}`}>
          {archKeys.map((arch) => {
            const d = downloads[arch];
            const isRecommended = archKeys.length > 1 && arch === recommended;
            return (
              <div
                key={arch}
                className={`bg-card rounded-xl p-6 shadow-lg transition-all duration-200 hover:shadow-xl ${
                  isRecommended
                    ? "border-2 border-indigo-400 relative"
                    : "border border-border"
                }`}
              >
                {isRecommended && (
                  <div className="absolute -top-3 left-1/2 -translate-x-1/2 px-3 py-1 bg-indigo-500 text-white text-xs font-semibold rounded-full">
                    推荐
                  </div>
                )}
                <div className="flex items-center gap-3 mb-4">
                  <div className={`w-12 h-12 rounded-lg flex items-center justify-center ${
                    isRecommended ? "bg-indigo-100" : "bg-blue-100"
                  }`}>
                    {platform === "macos" && arch === "arm64"
                      ? <Apple size={24} className="text-indigo-600" />
                      : <Cpu size={24} className="text-blue-600" />
                    }
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-foreground">{d.label}</h3>
                    <p className="text-sm text-muted-foreground">{d.chip}</p>
                  </div>
                </div>

                <div className="flex items-center justify-between mb-4">
                  <span className="text-xs text-muted-foreground font-mono">{d.file}</span>
                  <span className="text-xs text-muted-foreground">{d.size}</span>
                </div>

                <a
                  href={d.url}
                  download
                  className={`w-full py-3 font-semibold rounded-lg transition-colors inline-flex items-center justify-center gap-2 ${
                    isRecommended || archKeys.length === 1
                      ? "bg-indigo-500 hover:bg-indigo-600 text-white"
                      : "bg-primary hover:bg-primary/90 text-primary-foreground"
                  }`}
                >
                  <Download size={18} />
                  {platform === "macos" ? "下载 DMG" : "下载安装包"}
                </a>
              </div>
            );
          })}
        </div>
      </div>

      {/* Features */}
      <div className="max-w-4xl mx-auto px-6 py-16">
        <h2 className="text-2xl font-bold text-center text-foreground mb-10">功能特性</h2>
        <div className="grid md:grid-cols-3 gap-6">
          {[
            { title: "一键配置", desc: "自动检测已安装的 AI 工具，一键完成 API 配置" },
            { title: "独立模型设置", desc: "每个工具可独立选择默认模型，灵活管理" },
            { title: "系统托盘常驻", desc: "后台运行，余额不足自动通知，不影响工作" },
          ].map((f) => (
            <div key={f.title} className="flex items-start gap-3">
              <div className="mt-1">
                <Check size={18} className="text-emerald-500" />
              </div>
              <div>
                <h3 className="font-semibold text-foreground mb-1">{f.title}</h3>
                <p className="text-sm text-muted-foreground">{f.desc}</p>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Install steps */}
      <div className="bg-slate-50 dark:bg-slate-900/50 py-16">
        <div className="max-w-4xl mx-auto px-6">
          <h2 className="text-2xl font-bold text-center text-foreground mb-10">安装步骤</h2>
          <div className="grid md:grid-cols-4 gap-8">
            {INSTALL_STEPS[platform].map((s) => (
              <div key={s.step} className="text-center">
                <div className="w-12 h-12 bg-indigo-100 dark:bg-indigo-900/30 rounded-full flex items-center justify-center mx-auto mb-3 text-indigo-600 dark:text-indigo-400 font-bold text-lg">
                  {s.step}
                </div>
                <h3 className="font-semibold text-foreground mb-2">{s.title}</h3>
                <p className="text-sm text-muted-foreground">{s.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* macOS security notice */}
      {platform === "macos" && (
        <div className="max-w-4xl mx-auto px-6 py-12">
          <div className="bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800 rounded-xl p-6">
            <div className="flex items-start gap-4">
              <div className="w-10 h-10 bg-amber-100 dark:bg-amber-900/50 rounded-lg flex items-center justify-center flex-shrink-0 mt-0.5">
                <ShieldAlert size={20} className="text-amber-600 dark:text-amber-400" />
              </div>
              <div className="flex-1 min-w-0">
                <h3 className="font-semibold text-amber-900 dark:text-amber-200 mb-2">
                  macOS 安全提示
                </h3>
                <p className="text-sm text-amber-800 dark:text-amber-300 mb-4">
                  首次打开时，macOS 可能提示"应用已损坏"或"无法验证开发者"。这是因为应用尚未经过 Apple 公证签名。
                  请在<strong>终端</strong>中执行以下命令解除限制，只需执行一次：
                </p>
                <div className="flex items-center gap-2 bg-slate-900 dark:bg-slate-950 rounded-lg p-3">
                  <Terminal size={16} className="text-slate-400 flex-shrink-0" />
                  <code className="text-sm text-emerald-400 font-mono flex-1 select-all overflow-x-auto">
                    {xattrCmd}
                  </code>
                  <button
                    onClick={handleCopy}
                    className="text-slate-400 hover:text-white transition-colors flex-shrink-0 p-1"
                    title="复制命令"
                  >
                    {copied ? <Check size={16} className="text-emerald-400" /> : <Copy size={16} />}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Windows security notice */}
      {platform === "windows" && (
        <div className="max-w-4xl mx-auto px-6 py-12">
          <div className="bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800 rounded-xl p-6">
            <div className="flex items-start gap-4">
              <div className="w-10 h-10 bg-amber-100 dark:bg-amber-900/50 rounded-lg flex items-center justify-center flex-shrink-0 mt-0.5">
                <ShieldAlert size={20} className="text-amber-600 dark:text-amber-400" />
              </div>
              <div className="flex-1 min-w-0">
                <h3 className="font-semibold text-amber-900 dark:text-amber-200 mb-2">
                  Windows 安全提示
                </h3>
                <p className="text-sm text-amber-800 dark:text-amber-300">
                  首次运行时，Windows SmartScreen 可能提示"Windows 已保护你的电脑"。
                  这是因为应用尚未经过代码签名。点击<strong>「详细信息」</strong>，然后点击<strong>「仍要运行」</strong>即可正常使用。
                </p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* CTA */}
      <div className="max-w-4xl mx-auto px-6 py-16 text-center">
        <h2 className="text-2xl font-bold text-foreground mb-4">还没有账号？</h2>
        <p className="text-muted-foreground mb-6">注册即送 ¥5 试用额度，下载 Desktop 后一键配置</p>
        <div className="flex gap-4 justify-center">
          <button
            onClick={() => navigate("/auth?tab=register")}
            className="px-8 py-3 bg-gradient-to-r from-indigo-500 to-indigo-600 hover:from-indigo-600 hover:to-indigo-700 text-white font-semibold rounded-xl shadow-lg hover:shadow-xl transition-all duration-200 flex items-center gap-2"
          >
            免费注册
            <ArrowRight size={18} />
          </button>
          <button
            onClick={() => navigate("/docs")}
            className="px-8 py-3 bg-card hover:bg-muted text-foreground font-semibold rounded-xl border border-border transition-all duration-200"
          >
            查看文档
          </button>
        </div>
      </div>

      {/* Footer */}
      <div className="border-t border-border py-8">
        <div className="max-w-5xl mx-auto px-6 text-center text-sm text-muted-foreground">
          <div className="flex items-center justify-center gap-2 mb-2">
            <Zap size={16} className="text-primary" />
            <span className="font-semibold text-foreground">LLM Gateway</span>
          </div>
          <p>国内 AI 编码工具统一网关 · 支持 Claude Code / Codex CLI / OpenClaw / Hermes</p>
          <div className="mt-2">
            <BeianBar />
          </div>
        </div>
      </div>
    </div>
  );
};

export default DownloadPage;
