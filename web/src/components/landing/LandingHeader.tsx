import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Zap, Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";
import { useAuth } from "@/contexts/AuthContext";

const LandingHeader = () => {
  const navigate = useNavigate();
  const auth = useAuth();
  const { theme, setTheme } = useTheme();
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 8);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  return (
    <header
      className={`sticky top-0 z-50 w-full transition-all duration-200 ${
        scrolled
          ? "bg-background/80 glass border-b border-border/60"
          : "bg-transparent"
      }`}
    >
      <div className="max-w-6xl mx-auto px-6 h-14 flex items-center">
        <Link
          to="/"
          className="flex items-center gap-2.5 select-none mr-auto"
        >
          <div className="w-8 h-8 brand-gradient rounded-xl flex items-center justify-center shadow-button">
            <Zap size={15} className="text-white" />
          </div>
          <span className="font-bold text-sm tracking-tight">LLM Gateway</span>
        </Link>

        <nav className="hidden md:flex items-center gap-6 mr-6 text-sm text-muted-foreground">
          <a href="#features" className="hover:text-foreground transition-colors">功能</a>
          <a href="#models" className="hover:text-foreground transition-colors">模型</a>
          <a href="#plans" className="hover:text-foreground transition-colors">套餐</a>
          <a href="#faq" className="hover:text-foreground transition-colors">常见问题</a>
          <Link to="/enterprise" className="hover:text-foreground transition-colors">企业版</Link>
          <Link to="/docs" className="hover:text-foreground transition-colors">文档</Link>
        </nav>

        <button
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
          className="w-8 h-8 mr-2 flex items-center justify-center rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted/60 transition-colors duration-150 cursor-pointer"
          title={theme === "dark" ? "切换到浅色模式" : "切换到深色模式"}
        >
          {theme === "dark" ? <Sun size={16} /> : <Moon size={16} />}
        </button>

        {auth.isAuthenticated ? (
          <button
            onClick={() => navigate("/dashboard")}
            className="px-4 py-1.5 text-sm font-semibold rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
          >
            进入控制台
          </button>
        ) : (
          <div className="flex items-center gap-2">
            <button
              onClick={() => navigate("/login")}
              className="px-3 py-1.5 text-sm rounded-lg text-foreground hover:bg-muted/60 transition-colors"
            >
              登录
            </button>
            <button
              onClick={() => navigate("/auth?tab=register")}
              className="px-4 py-1.5 text-sm font-semibold rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              免费注册
            </button>
          </div>
        )}
      </div>
    </header>
  );
};

export default LandingHeader;
