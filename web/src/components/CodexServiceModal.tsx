import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Copy, Check } from "lucide-react";
import { useState } from "react";

interface CodexServiceModalProps {
  isOpen: boolean;
  onClose: () => void;
  wechatId: string;
}

export default function CodexServiceModal({ isOpen, onClose, wechatId }: CodexServiceModalProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(wechatId);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>支付成功，请添加客服微信</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="text-center">
            <p className="text-sm text-muted-foreground mb-4">
              请添加客服微信，并提供您的订单号以便我们为您发货
            </p>
            <div className="flex items-center justify-center gap-2 p-4 bg-muted rounded-lg">
              <span className="text-lg font-mono font-bold">{wechatId}</span>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleCopy}
                className="h-8 w-8 p-0"
              >
                {copied ? (
                  <Check className="h-4 w-4 text-green-600" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>
          <div className="text-xs text-muted-foreground text-center">
            <p>提示：复制微信号后，打开微信添加好友</p>
          </div>
          <Button onClick={onClose} className="w-full">
            知道了
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
