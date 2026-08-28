import { Link } from "react-router-dom";
import { SITE } from "@/config/site";

const BeianBar = ({ className = "" }: { className?: string }) => (
  <span
    className={`inline-flex flex-wrap items-center justify-center gap-x-2 gap-y-1 text-xs text-muted-foreground ${className}`}
  >
    <a
      href={SITE.icpUrl}
      target="_blank"
      rel="noreferrer"
      className="hover:text-foreground transition-colors"
    >
      {SITE.icpNumber}
    </a>
    <span aria-hidden="true">|</span>
    <Link
      to={SITE.qualificationsPath}
      className="hover:text-foreground transition-colors"
    >
      增值电信业务经营许可证：{SITE.telecomLicenseNumber}
    </Link>
  </span>
);

export default BeianBar;
