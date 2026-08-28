import { useNavigate } from "react-router-dom";
import { LogOut, Image as ImageIcon } from "lucide-react";
import type { ReactNode } from "react";
import { useAuth } from "@/contexts/AuthContext";
import { isImageHost } from "@/lib/host";

interface Props {
  children: ReactNode;
}

const ImageShareLayout = ({ children }: Props) => {
  const auth = useAuth();
  const navigate = useNavigate();
  const share = auth.imageShare;

  const handleLogout = async () => {
    await auth.logout();
    navigate(isImageHost() ? "/login" : "/image-login", { replace: true });
  };

  const remaining = share?.quota_remaining ?? 0;
  const total = share?.quota_total ?? 0;
  const used = share?.quota_used ?? 0;

  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="sticky top-0 z-10 flex items-center justify-between border-b bg-card/80 px-4 py-3 backdrop-blur sm:px-6">
        <div className="flex items-center gap-2">
          <ImageIcon className="h-5 w-5 text-primary" />
          <span className="font-medium">图片生成</span>
          {share?.name && (
            <span className="ml-2 hidden rounded-md bg-muted px-2 py-0.5 text-xs text-muted-foreground sm:inline">{share.name}</span>
          )}
        </div>
        <div className="flex items-center gap-3 text-sm">
          <div className="rounded-md border bg-card px-3 py-1.5">
            <span className="text-muted-foreground">剩余配额：</span>
            <span className="font-semibold tabular-nums text-primary">{remaining}</span>
            <span className="text-muted-foreground"> / {total} 张</span>
          </div>
          <button
            onClick={handleLogout}
            className="inline-flex items-center gap-1 rounded-md border px-3 py-1.5 text-sm transition hover:bg-muted"
            title="退出"
          >
            <LogOut className="h-4 w-4" />
            <span className="hidden sm:inline">退出</span>
          </button>
        </div>
      </header>
      <main className="flex-1">{children}</main>
      <footer className="border-t px-6 py-2 text-center text-xs text-muted-foreground">
        累计已生成 {used} 张
      </footer>
    </div>
  );
};

export default ImageShareLayout;
