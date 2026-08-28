import { Link } from "react-router-dom";
import { Zap } from "lucide-react";
import BeianBar from "@/components/BeianBar";

const LandingFooter = () => {
  return (
    <footer className="border-t border-border bg-background">
      <div className="max-w-6xl mx-auto px-6 py-12 grid gap-8 md:grid-cols-4">
        <div className="md:col-span-2">
          <div className="flex items-center gap-2 mb-3">
            <div className="w-7 h-7 brand-gradient rounded-lg flex items-center justify-center shadow-button">
              <Zap size={13} className="text-white" />
            </div>
            <span className="font-bold text-foreground">LLM Gateway</span>
          </div>
          <p className="text-sm text-muted-foreground leading-relaxed max-w-md">
            统一的大模型 API 网关，一个账号接入国内外主流大模型。
            原生协议透传，按量计费，支持团队与企业账户。
          </p>
        </div>

        <div>
          <h4 className="text-sm font-semibold text-foreground mb-3">产品</h4>
          <ul className="space-y-2 text-sm text-muted-foreground">
            <li><a href="#features" className="hover:text-foreground transition-colors">核心功能</a></li>
            <li><a href="#models" className="hover:text-foreground transition-colors">支持模型</a></li>
            <li><a href="#plans" className="hover:text-foreground transition-colors">订阅套餐</a></li>
            <li><Link to="/claude" className="hover:text-foreground transition-colors">Claude 专区</Link></li>
          </ul>
        </div>

        <div>
          <h4 className="text-sm font-semibold text-foreground mb-3">资源</h4>
          <ul className="space-y-2 text-sm text-muted-foreground">
            <li><Link to="/docs" className="hover:text-foreground transition-colors">API 文档</Link></li>
            <li><Link to="/download" className="hover:text-foreground transition-colors">下载客户端</Link></li>
            <li><Link to="/login" className="hover:text-foreground transition-colors">登录</Link></li>
            <li><Link to="/auth?tab=register" className="hover:text-foreground transition-colors">注册账号</Link></li>
          </ul>
        </div>
      </div>

      <div className="border-t border-border">
        <div className="max-w-6xl mx-auto px-6 py-5 flex flex-col md:flex-row items-center justify-between gap-3 text-xs text-muted-foreground">
          <div className="flex items-center gap-x-3 gap-y-1 flex-wrap">
            <span>© {new Date().getFullYear()} LLM Gateway · 统一的大模型 API 网关</span>
            <BeianBar />
          </div>
          <div className="flex items-center gap-2">
            <img src="/wechat_QR.png" alt="客服微信二维码" className="w-12 h-12 rounded" />
            <span>微信扫码联系客服</span>
          </div>
        </div>
      </div>
    </footer>
  );
};

export default LandingFooter;
