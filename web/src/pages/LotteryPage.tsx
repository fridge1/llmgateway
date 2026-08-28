import { useState } from "react";
import { Gift, Trophy, Clock, ChevronLeft, ChevronRight, Loader2, Ticket } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";
import { useLotteryCurrentEvent, useLotteryWinnerRecords, useLotteryMyRecords } from "@/hooks/use-api";
import type { LotteryPrize, PublicLotteryRecord, LotteryRecord } from "@/types/api";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { PageHeader } from "@/components/ui/page-header";
function fmtDate(s: string) {
  return new Date(s).toLocaleString("zh-CN", { dateStyle: "short", timeStyle: "short" });
}

function fmtCny(n: number) {
  return `¥${n.toFixed(2)}`;
}

function PrizeCard({ prize }: { prize: LotteryPrize }) {
  const isReward = prize.prize_type === "balance" || prize.prize_type === "match_recharge";
  return (
    <div className={`rounded-xl border p-4 flex flex-col gap-1 ${isReward ? "border-primary/30 bg-primary/5" : "border-border bg-card"}`}>
      <div className="flex items-center gap-2">
        <Gift size={16} className={isReward ? "text-primary" : "text-muted-foreground"} />
        <span className="font-semibold text-sm">{prize.name}</span>
      </div>
      {prize.description && <p className="text-xs text-muted-foreground">{prize.description}</p>}
      <div className="mt-1 text-xs font-medium text-muted-foreground">
        {prize.prize_type === "balance" && <span className="text-emerald-600">奖励 {fmtCny(prize.prize_value)}</span>}
        {prize.prize_type === "match_recharge" && <span className="text-emerald-600">奖励与充值额相同</span>}
        {prize.prize_type === "physical" && <span>实物奖品，人工跟进</span>}
        {prize.prize_type === "none" && <span>感谢参与</span>}
      </div>
      {prize.total_stock > 0 && (
        <div className="text-xs text-muted-foreground">剩余 {prize.remaining_stock} / {prize.total_stock}</div>
      )}
    </div>
  );
}

function RecordRow({ r }: { r: PublicLotteryRecord }) {
  const won = r.prize_type !== "none" && r.prize_name;
  return (
    <TableRow className="border-t border-border hover:bg-muted/20">
      <TableCell className="px-4 py-3 text-sm">{fmtDate(r.created_at)}</TableCell>
      <TableCell className="px-4 py-3 text-sm">{r.masked_phone}</TableCell>
      <TableCell className="px-4 py-3 text-sm">{r.prize_name || "—"}</TableCell>
      <TableCell className="px-4 py-3 text-sm">
        {(r.prize_type === "balance" || r.prize_type === "match_recharge") && r.prize_value > 0
          ? <span className="text-emerald-600 font-medium">{fmtCny(r.prize_value)} 到账</span>
          : won ? <span className="text-muted-foreground">人工处理</span> : <span className="text-muted-foreground">—</span>
        }
      </TableCell>
    </TableRow>
  );
}

export default function LotteryPage() {
  const [page, setPage] = useState(1);
  const { data: eventData, isLoading: eventLoading } = useLotteryCurrentEvent();
  const { data: recordsData, isLoading: recordsLoading } = useLotteryWinnerRecords(page, 10);

  const event = eventData?.event;
  const prizes = eventData?.prizes ?? [];
  const records = recordsData?.records ?? [];
  const total = recordsData?.total ?? 0;
  const totalPages = Math.ceil(total / 10);

  return (
    <div className="page-container max-w-5xl space-y-6 fade-in">
      <PageHeader
        eyebrow="福利活动"
        title={<span className="flex items-center gap-2.5"><Trophy size={20} className="text-primary" />抽奖活动</span>}
        description="充值达到门槛即自动参与抽奖，活动开奖后奖励直接到账。"
      />

      {/* Current event */}
      {eventLoading ? (
        <div className="flex justify-center py-10"><Loader2 size={20} className="animate-spin text-muted-foreground" /></div>
      ) : !event ? (
        <div className="rounded-xl border border-border bg-card p-8 text-center text-muted-foreground text-sm">
          暂无进行中的抽奖活动
        </div>
      ) : (
        <div className="rounded-xl border border-border bg-card overflow-hidden">
          <div className="px-6 py-5 border-b border-border bg-primary/5">
            <div className="flex items-start justify-between gap-4">
              <div>
                <h2 className="text-lg font-semibold">{event.name}</h2>
                {event.description && <p className="text-sm text-muted-foreground mt-1">{event.description}</p>}
              </div>
              <span className={`shrink-0 text-xs font-medium px-2.5 py-1 rounded-full ${
                event.status === "active" ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400" : "bg-muted text-muted-foreground"
              }`}>
                {event.status === "active" ? "进行中" : event.status === "paused" ? "已暂停" : "已结束"}
              </span>
            </div>
            <div className="mt-3 flex flex-wrap gap-4 text-sm text-muted-foreground">
              <span>最低充值 <span className="font-semibold text-foreground">{fmtCny(event.min_recharge_cny)}</span> 即可参与</span>
              <span>已有 <span className="font-semibold text-foreground">{event.participant_count}</span> 人参与</span>
              {event.start_time && (
                <span className="flex items-center gap-1"><Clock size={13} />开始：{fmtDate(event.start_time)}</span>
              )}
              {event.end_time && (
                <span className="flex items-center gap-1"><Clock size={13} />截止：{fmtDate(event.end_time)}</span>
              )}
            </div>
          </div>
          <div className="px-6 py-5">
            <h3 className="text-sm font-semibold mb-3">奖品列表</h3>
            {prizes.length === 0 ? (
              <p className="text-sm text-muted-foreground">暂无奖品配置</p>
            ) : (
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                {prizes.map(p => <PrizeCard key={p.id} prize={p} />)}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Draw records */}
      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="px-6 py-4 border-b border-border">
          <h2 className="text-base font-semibold">中奖记录</h2>
        </div>
        {recordsLoading ? (
          <div className="flex justify-center py-8"><Loader2 size={18} className="animate-spin text-muted-foreground" /></div>
        ) : records.length === 0 ? (
          <div className="py-8 text-center text-sm text-muted-foreground">暂无中奖记录</div>
        ) : (
          <>
            <Table className="w-full text-sm">
              <TableHeader>
                <TableRow className="text-left text-xs text-muted-foreground bg-muted/30">
                  <TableHead className="px-4 py-2 font-medium">中奖时间</TableHead>
                  <TableHead className="px-4 py-2 font-medium">中奖用户</TableHead>
                  <TableHead className="px-4 py-2 font-medium">奖品</TableHead>
                  <TableHead className="px-4 py-2 font-medium">奖励</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {records.map(r => <RecordRow key={r.id} r={r} />)}
              </TableBody>
            </Table>
            {totalPages > 1 && (
              <div className="flex items-center justify-center gap-2 px-4 py-3 border-t border-border">
                <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1} className="flex h-9 w-9 items-center justify-center rounded-lg hover:bg-muted disabled:opacity-40" aria-label="上一页">
                  <ChevronLeft size={16} />
                </button>
                <span className="text-sm text-muted-foreground">{page} / {totalPages}</span>
                <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages} className="flex h-9 w-9 items-center justify-center rounded-lg hover:bg-muted disabled:opacity-40" aria-label="下一页">
                  <ChevronRight size={16} />
                </button>
              </div>
            )}
          </>
        )}
      </div>

      {/* 我的参与记录（仅登录用户可见） */}
      <MyLotteryRecords />
    </div>
  );
}

function prizeTypeLabel(t: string) {
  if (t === "none") return "谢谢参与";
  if (t === "balance") return "余额奖励";
  if (t === "match_recharge") return "等额充值";
  if (t === "physical") return "实物奖品";
  return t;
}

function MyLotteryRecords() {
  const { user } = useAuth();
  const [page, setPage] = useState(1);
  const { data, isLoading } = useLotteryMyRecords(page, 10);
  if (!user) return null;

  const records = data?.records ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / 10);

  return (
    <div className="rounded-xl border border-border bg-card overflow-hidden">
      <div className="px-6 py-4 border-b border-border flex items-center gap-2">
        <Ticket size={16} className="text-primary" />
        <h2 className="text-base font-semibold">我的参与记录</h2>
      </div>
      {isLoading ? (
        <div className="flex justify-center py-8"><Loader2 size={18} className="animate-spin text-muted-foreground" /></div>
      ) : records.length === 0 ? (
        <div className="py-8 text-center text-sm text-muted-foreground">暂无参与记录，充值达到门槛后将自动参与</div>
      ) : (
        <>
          <Table className="w-full text-sm">
            <TableHeader>
              <TableRow className="text-left text-xs text-muted-foreground bg-muted/30">
                <TableHead className="px-4 py-2 font-medium">参与时间</TableHead>
                <TableHead className="px-4 py-2 font-medium">充值金额</TableHead>
                <TableHead className="px-4 py-2 font-medium">抽奖结果</TableHead>
                <TableHead className="px-4 py-2 font-medium">奖励</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {records.map((r: LotteryRecord) => {
                const won = r.prize_id != null && r.prize_type !== "none";
                return (
                  <TableRow key={r.id} className="border-t border-border hover:bg-muted/20">
                    <TableCell className="px-4 py-3 text-sm">{fmtDate(r.created_at)}</TableCell>
                    <TableCell className="px-4 py-3 text-sm">{fmtCny(r.recharge_amount)}</TableCell>
                    <TableCell className="px-4 py-3 text-sm">{won ? r.prize_name : "未开奖 / 谢谢参与"}</TableCell>
                    <TableCell className="px-4 py-3 text-sm">
                      {(r.prize_type === "balance" || r.prize_type === "match_recharge") && r.prize_value > 0
                        ? <span className="text-emerald-600 font-medium">{fmtCny(r.prize_value)} 到账</span>
                        : won
                          ? <span className="text-muted-foreground">{prizeTypeLabel(r.prize_type)}</span>
                          : <span className="text-muted-foreground">—</span>}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 px-4 py-3 border-t border-border">
              <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1} className="flex h-9 w-9 items-center justify-center rounded-lg hover:bg-muted disabled:opacity-40" aria-label="上一页">
                <ChevronLeft size={16} />
              </button>
              <span className="text-sm text-muted-foreground">{page} / {totalPages}</span>
              <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages} className="flex h-9 w-9 items-center justify-center rounded-lg hover:bg-muted disabled:opacity-40" aria-label="下一页">
                <ChevronRight size={16} />
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
