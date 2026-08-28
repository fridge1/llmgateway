import type { LucideIcon } from "lucide-react";
import {
  MessageSquare,
  Languages,
  FileText,
  Image,
  Sparkles,
  Presentation,
  Sheet,
} from "lucide-react";

export interface ToolDefinition {
  id: string;
  name: string;
  description: string;
  icon: LucideIcon;
  route: string;
  status: "available" | "coming_soon";
  color: string;
}

export const tools: ToolDefinition[] = [
  {
    id: "chat",
    name: "AI 对话",
    description: "多模型对话体验，支持单模型与模型对比模式",
    icon: MessageSquare,
    route: "chat",
    status: "available",
    color: "bg-primary",
  },
  {
    id: "translate",
    name: "文本翻译",
    description: "基于大模型的智能多语言翻译",
    icon: Languages,
    route: "translate",
    status: "coming_soon",
    color: "bg-emerald-500",
  },
  {
    id: "summary",
    name: "文档摘要",
    description: "快速提取文档核心内容与关键信息",
    icon: FileText,
    route: "summary",
    status: "coming_soon",
    color: "bg-orange-500",
  },
  {
    id: "image",
    name: "图片生成",
    description: "基于文本描述生成高质量图片",
    icon: Image,
    route: "image",
    status: "available",
    color: "bg-pink-500",
  },
  {
    id: "prompt",
    name: "Prompt 优化器",
    description: "优化你的 Prompt，减少 token 用量、提升输出质量",
    icon: Sparkles,
    route: "prompt",
    status: "coming_soon",
    color: "bg-violet-500",
  },
  {
    id: "ppt",
    name: "PPT 生成",
    description: "AI 一键生成演示文稿，支持自定义主题与排版",
    icon: Presentation,
    route: "ppt",
    status: "coming_soon",
    color: "bg-blue-500",
  },
  {
    id: "excel",
    name: "Excel 智能处理",
    description: "AI 辅助表格数据分析、清洗与公式生成",
    icon: Sheet,
    route: "excel",
    status: "coming_soon",
    color: "bg-green-500",
  },
];
