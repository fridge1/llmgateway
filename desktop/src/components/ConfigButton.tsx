import { CheckCircle2, Wrench, Loader2, RefreshCw } from "./icons";

interface Props {
  unconfiguredCount: number;
  totalCount: number;
  configuring: boolean;
  onConfigure: () => void;
  onReconfigure: () => void;
}

export default function ConfigButton({ unconfiguredCount, totalCount: _totalCount, configuring, onConfigure, onReconfigure }: Props) {
  if (unconfiguredCount === 0) {
    return (
      <div className="flex items-center justify-between py-3">
        <div className="flex items-center gap-2">
          <CheckCircle2 size={18} className="text-emerald-400" />
          <span className="text-emerald-400 text-sm">所有工具已配置完成</span>
        </div>
        <button
          onClick={onReconfigure}
          disabled={configuring}
          className="px-3 py-1.5 text-xs text-obsidian-400 hover:text-obsidian-200 border border-obsidian-700 hover:border-obsidian-500 disabled:text-obsidian-600 rounded-lg transition-colors inline-flex items-center gap-1.5"
        >
          {configuring ? (
            <><Loader2 size={12} className="animate-spin" /> 配置中...</>
          ) : (
            <><RefreshCw size={12} /> 重新配置</>
          )}
        </button>
      </div>
    );
  }

  return (
    <button
      onClick={onConfigure}
      disabled={configuring}
      className="w-full py-3 bg-amber-500 hover:bg-amber-400 hover:shadow-glow-amber disabled:bg-obsidian-800 disabled:text-obsidian-600 text-obsidian-950 rounded-xl font-semibold transition-all duration-200 inline-flex items-center justify-center gap-2"
    >
      {configuring ? (
        <>
          <Loader2 size={16} className="animate-spin" />
          配置中...
        </>
      ) : (
        <>
          <Wrench size={16} />
          一键配置 {unconfiguredCount} 个工具
        </>
      )}
    </button>
  );
}
