import type { LucideIcon } from "lucide-react";
import { useNavigate } from "react-router-dom";
import type { ReactNode } from "react";
import { PageHeader } from "@/components/ui/page-header";

interface TenantPageHeaderProps {
  title: string;
  description?: string;
  tenantName?: string;
  icon: LucideIcon;
  backTo?: string;
  onBack?: () => void;
  actions?: ReactNode;
}

export function TenantPageHeader({
  title,
  description,
  tenantName,
  icon: Icon,
  backTo,
  onBack,
  actions,
}: TenantPageHeaderProps) {
  const navigate = useNavigate();
  const handleBack = onBack ?? (backTo ? () => navigate(backTo) : undefined);

  return (
    <PageHeader
      eyebrow={tenantName ? `组织 / ${tenantName}` : "组织"}
      title={
        <span className="flex items-center gap-2.5">
          <Icon size={20} className="text-primary" />
          {title}
        </span>
      }
      description={description}
      backAction={handleBack}
      actions={actions}
    />
  );
}
