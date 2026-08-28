import { useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { Image as ImageIcon, Key } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";
import { isImageHost } from "@/lib/host";

const ImageShareLoginPage = () => {
  const navigate = useNavigate();
  const auth = useAuth();
  const [key, setKey] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  if (auth.isLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    );
  }

  if (auth.isImageShare) {
    return <Navigate to="/image" replace />;
  }
  // Only redirect a regular user to the dashboard on the main host.
  // The image-share subdomain has no /dashboard route.
  if (auth.user && !isImageHost()) {
    return <Navigate to="/dashboard" replace />;
  }

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (loading || !key.trim()) return;
    setError("");
    setLoading(true);
    try {
      await auth.loginImageShare(key.trim());
      navigate("/image", { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : "密钥无效");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-background to-muted px-4">
      <div className="w-full max-w-md rounded-2xl border bg-card p-8 shadow-lg">
        <div className="mb-6 flex flex-col items-center text-center">
          <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-primary/10 text-primary">
            <ImageIcon className="h-6 w-6" />
          </div>
          <h1 className="text-xl font-semibold">图片生成密钥登录</h1>
          <p className="mt-1 text-sm text-muted-foreground">输入分发的密钥后即可开始生成图片</p>
        </div>

        <form onSubmit={submit} className="space-y-4">
          <div className="relative">
            <Key className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              autoFocus
              autoComplete="off"
              spellCheck={false}
              placeholder="sk-img-..."
              value={key}
              onChange={(e) => setKey(e.target.value)}
              className="w-full rounded-md border bg-background py-2 pl-9 pr-3 text-sm font-mono tracking-tight outline-none focus:border-primary"
            />
          </div>
          {error && (
            <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</div>
          )}
          <button
            type="submit"
            disabled={loading || !key.trim()}
            className="w-full rounded-md bg-primary py-2 text-sm font-medium text-primary-foreground transition hover:bg-primary/90 disabled:opacity-50"
          >
            {loading ? "登录中…" : "登录"}
          </button>
        </form>
      </div>
    </div>
  );
};

export default ImageShareLoginPage;
