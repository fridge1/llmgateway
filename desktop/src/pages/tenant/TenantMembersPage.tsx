import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  useTenantMembers, useInviteMember, useRemoveMember, useUpdateMemberRole,
  useTransferOwnership, useTenantSubUsers, useCreateSubUser, useDeleteSubUser,
  useResetSubUserPassword, useUpdateSubUserQuota,
} from "@/hooks/use-tenant";
import { Loader2 } from "../../components/icons";

export default function TenantMembersPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { data: members = [], isLoading } = useTenantMembers(id);
  const { data: subUsers = [] } = useTenantSubUsers(id);
  const inviteMember = useInviteMember(id);
  const removeMember = useRemoveMember(id);
  const updateRole = useUpdateMemberRole(id);
  const transferOwnership = useTransferOwnership(id);
  const createSubUser = useCreateSubUser(id);
  const deleteSubUser = useDeleteSubUser(id);
  const resetPassword = useResetSubUserPassword(id);
  const updateQuota = useUpdateSubUserQuota(id);

  const [showInvite, setShowInvite] = useState(false);
  const [invitePhone, setInvitePhone] = useState("");
  const [inviteRole, setInviteRole] = useState("member");
  const [showSubUser, setShowSubUser] = useState(false);
  const [subForm, setSubForm] = useState({ username: "", password: "", nickname: "", quota_limit: "" });
  const [tab, setTab] = useState<"members" | "subusers">("members");

  const handleInvite = async () => {
    if (!invitePhone.trim()) return;
    try {
      await inviteMember.mutateAsync({ phone: invitePhone.trim(), role: inviteRole });
      setShowInvite(false);
      setInvitePhone("");
    } catch {}
  };

  const handleCreateSubUser = async () => {
    if (!subForm.username.trim() || !subForm.password.trim()) return;
    try {
      await createSubUser.mutateAsync({
        username: subForm.username.trim(),
        password: subForm.password.trim(),
        nickname: subForm.nickname || undefined,
        quota_limit: subForm.quota_limit ? Number(subForm.quota_limit) : null,
      });
      setShowSubUser(false);
      setSubForm({ username: "", password: "", nickname: "", quota_limit: "" });
    } catch {}
  };

  if (isLoading) {
    return (
      <div className="p-6 flex items-center justify-center" style={{ minHeight: 400 }}>
        <Loader2 size={24} className="animate-spin text-amber-400" />
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="flex items-center gap-2 mb-5">
        <button onClick={() => navigate(`/tenants/${id}`)} className="text-xs text-obsidian-400 hover:text-obsidian-200">← 返回</button>
        <h1 className="text-lg font-semibold text-obsidian-50">成员管理</h1>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-obsidian-800 rounded-lg p-0.5 mb-5 w-fit">
        <button onClick={() => setTab("members")} className={`px-4 py-1.5 rounded-md text-xs font-medium transition-all ${tab === "members" ? "bg-amber-500 text-obsidian-950" : "text-obsidian-400"}`}>
          成员 ({members.length})
        </button>
        <button onClick={() => setTab("subusers")} className={`px-4 py-1.5 rounded-md text-xs font-medium transition-all ${tab === "subusers" ? "bg-amber-500 text-obsidian-950" : "text-obsidian-400"}`}>
          子用户 ({subUsers.length})
        </button>
      </div>

      {tab === "members" ? (
        <>
          <div className="flex justify-end mb-3">
            <button onClick={() => setShowInvite(true)} className="px-4 py-2 bg-amber-500 hover:bg-amber-400 text-obsidian-950 text-sm font-semibold rounded-lg transition-all duration-200">
              邀请成员
            </button>
          </div>
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="text-xs text-obsidian-500 border-b border-obsidian-800">
                  <th className="text-left px-4 py-2 font-medium">手机号</th>
                  <th className="text-left px-4 py-2 font-medium">角色</th>
                  <th className="text-left px-4 py-2 font-medium">加入时间</th>
                  <th className="text-right px-4 py-2 font-medium">操作</th>
                </tr>
              </thead>
              <tbody>
                {members.map((m) => (
                  <tr key={m.id} className="border-t border-obsidian-800 hover:bg-obsidian-800/50 transition-colors">
                    <td className="px-4 py-3 text-sm text-obsidian-200">{m.phone || m.user_id}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                        m.role === "owner" ? "bg-amber-500/10 text-amber-400" : "bg-obsidian-800 text-obsidian-300"
                      }`}>{m.role}</span>
                    </td>
                    <td className="px-4 py-3 text-xs text-obsidian-400">{new Date(m.joined_at).toLocaleDateString("zh-CN")}</td>
                    <td className="px-4 py-3 text-right">
                      {m.role !== "owner" && (
                        <div className="flex items-center justify-end gap-1">
                          <button
                            onClick={() => updateRole.mutate({ userId: m.user_id, role: m.role === "admin" ? "member" : "admin" })}
                            className="px-2 py-1 text-xs text-obsidian-400 hover:text-obsidian-200"
                          >{m.role === "admin" ? "降为成员" : "设为管理员"}</button>
                          <button
                            onClick={() => transferOwnership.mutate(m.user_id)}
                            className="px-2 py-1 text-xs text-obsidian-400 hover:text-amber-400"
                          >转让</button>
                          <button
                            onClick={() => window.confirm("确定移除此成员？") && removeMember.mutate(m.user_id)}
                            className="px-2 py-1 text-xs text-obsidian-400 hover:text-red-400"
                          >移除</button>
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      ) : (
        <>
          <div className="flex justify-end mb-3">
            <button onClick={() => setShowSubUser(true)} className="px-4 py-2 bg-amber-500 hover:bg-amber-400 text-obsidian-950 text-sm font-semibold rounded-lg transition-all duration-200">
              创建子用户
            </button>
          </div>
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="text-xs text-obsidian-500 border-b border-obsidian-800">
                  <th className="text-left px-4 py-2 font-medium">用户名</th>
                  <th className="text-left px-4 py-2 font-medium">昵称</th>
                  <th className="text-left px-4 py-2 font-medium">额度</th>
                  <th className="text-left px-4 py-2 font-medium">已用</th>
                  <th className="text-right px-4 py-2 font-medium">操作</th>
                </tr>
              </thead>
              <tbody>
                {subUsers.length === 0 ? (
                  <tr><td colSpan={5} className="px-4 py-12 text-center text-sm text-obsidian-500">暂无子用户</td></tr>
                ) : subUsers.map((u) => (
                  <tr key={u.id} className="border-t border-obsidian-800 hover:bg-obsidian-800/50 transition-colors">
                    <td className="px-4 py-3 text-sm text-obsidian-200">{u.username}</td>
                    <td className="px-4 py-3 text-xs text-obsidian-400">{u.nickname || "—"}</td>
                    <td className="px-4 py-3 text-xs text-obsidian-300">{u.quota_limit !== null ? `¥${u.quota_limit}` : "无限制"}</td>
                    <td className="px-4 py-3 text-xs text-obsidian-400">¥{u.quota_used.toFixed(2)}</td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={() => navigate(`/tenants/${id}/members/${u.id}/transactions`)}
                          className="px-2 py-1 text-xs text-obsidian-400 hover:text-obsidian-200"
                        >交易</button>
                        <button
                          onClick={() => {
                            const pw = window.prompt("输入新密码");
                            if (pw) resetPassword.mutate({ subUserId: u.id, password: pw });
                          }}
                          className="px-2 py-1 text-xs text-obsidian-400 hover:text-obsidian-200"
                        >重置密码</button>
                        <button
                          onClick={() => {
                            const q = window.prompt("输入额度限制（留空为无限制）");
                            if (q !== null) updateQuota.mutate({ subUserId: u.id, quota_limit: q ? Number(q) : null });
                          }}
                          className="px-2 py-1 text-xs text-obsidian-400 hover:text-amber-400"
                        >额度</button>
                        <button
                          onClick={() => window.confirm("确定删除此子用户？") && deleteSubUser.mutate(u.id)}
                          className="px-2 py-1 text-xs text-obsidian-400 hover:text-red-400"
                        >删除</button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {/* Invite modal */}
      {showInvite && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl w-[400px] p-6">
            <h3 className="text-base font-semibold text-obsidian-50 mb-4">邀请成员</h3>
            <div className="space-y-3 mb-4">
              <input className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50" placeholder="手机号" value={invitePhone} onChange={(e) => setInvitePhone(e.target.value)} />
              <select value={inviteRole} onChange={(e) => setInviteRole(e.target.value)} className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 focus:outline-none focus:border-amber-500/50">
                <option value="member">成员</option>
                <option value="admin">管理员</option>
              </select>
            </div>
            <div className="flex gap-3 justify-end">
              <button onClick={() => setShowInvite(false)} className="px-4 py-2 text-sm text-obsidian-300">取消</button>
              <button onClick={handleInvite} disabled={inviteMember.isPending} className="px-4 py-2 bg-amber-500 hover:bg-amber-400 disabled:bg-obsidian-800 text-obsidian-950 text-sm font-semibold rounded-lg transition-all">邀请</button>
            </div>
          </div>
        </div>
      )}

      {/* Create sub-user modal */}
      {showSubUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
          <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl w-[400px] p-6">
            <h3 className="text-base font-semibold text-obsidian-50 mb-4">创建子用户</h3>
            <div className="space-y-3 mb-4">
              <input className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50" placeholder="用户名" value={subForm.username} onChange={(e) => setSubForm({ ...subForm, username: e.target.value })} />
              <input className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50" type="password" placeholder="密码" value={subForm.password} onChange={(e) => setSubForm({ ...subForm, password: e.target.value })} />
              <input className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50" placeholder="昵称（可选）" value={subForm.nickname} onChange={(e) => setSubForm({ ...subForm, nickname: e.target.value })} />
              <input className="w-full px-3 py-2 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50" type="number" placeholder="额度限制（可选，留空为无限制）" value={subForm.quota_limit} onChange={(e) => setSubForm({ ...subForm, quota_limit: e.target.value })} />
            </div>
            <div className="flex gap-3 justify-end">
              <button onClick={() => setShowSubUser(false)} className="px-4 py-2 text-sm text-obsidian-300">取消</button>
              <button onClick={handleCreateSubUser} disabled={createSubUser.isPending} className="px-4 py-2 bg-amber-500 hover:bg-amber-400 disabled:bg-obsidian-800 text-obsidian-950 text-sm font-semibold rounded-lg transition-all">创建</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
