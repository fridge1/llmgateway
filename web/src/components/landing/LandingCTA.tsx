import { useNavigate } from "react-router-dom";
import { ArrowRight } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";

const LandingCTA = () => {
  const navigate = useNavigate();
  const auth = useAuth();

  if (auth.isAuthenticated) {
    return (
      <section className="py-20">
        <div className="max-w-4xl mx-auto px-6">
          <div className="relative overflow-hidden rounded-2xl balance-card-gradient p-12 text-center shadow-elevated">
            <div className="absolute inset-0 bg-grid-white/[0.08] bg-[size:40px_40px]" />
            <div className="relative">
              <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">
                继续探索控制台
              </h2>
              <p className="text-lg text-white/90 mb-8">
                查看用量、管理 API Key、调整订阅
              </p>
              <button
                onClick={() => navigate("/dashboard")}
                className="px-10 py-3.5 bg-white text-indigo-600 hover:bg-white/95 active:scale-[0.97] font-semibold rounded-xl shadow-lg transition-all inline-flex items-center gap-2 justify-center"
              >
                进入控制台
                <ArrowRight size={20} />
              </button>
            </div>
          </div>
        </div>
      </section>
    );
  }

  return (
    <section className="py-20">
      <div className="max-w-4xl mx-auto px-6">
        <div className="relative overflow-hidden rounded-2xl balance-card-gradient p-12 text-center shadow-elevated">
          <div className="absolute inset-0 bg-grid-white/[0.08] bg-[size:40px_40px]" />
          <div className="relative">
            <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">
              5 分钟，开始你的第一次调用
            </h2>
            <p className="text-lg text-white/90 mb-8">
              注册即送试用额度，无需信用卡
            </p>
            <div className="flex flex-col sm:flex-row gap-3 justify-center">
              <button
                onClick={() => navigate("/auth?tab=register")}
                className="px-10 py-3.5 bg-white text-indigo-600 hover:bg-white/95 font-semibold rounded-xl shadow-lg transition-all inline-flex items-center gap-2 justify-center"
              >
                免费注册
                <ArrowRight size={20} />
              </button>
              <button
                onClick={() => navigate("/docs")}
                className="px-10 py-3.5 bg-white/10 hover:bg-white/20 active:scale-[0.97] text-white font-semibold rounded-xl border border-white/20 backdrop-blur-sm transition-all flex items-center justify-center"
              >
                查看文档
              </button>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};

export default LandingCTA;
