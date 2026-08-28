# 支付宝证书目录

此目录用于存放支付宝支付所需的证书文件。**真实证书已被 `.gitignore` 忽略，请勿提交到代码仓库。**

如需启用支付宝支付，请将自己的证书放入此目录：

| 文件名 | 说明 |
|---|---|
| `app_private_key.pem` | 应用私钥（支付宝开放平台或商户后台生成） |
| `alipay_public_key.pem` | 支付宝公钥 |

可通过 `config.yaml` / `config.docker.yaml` 的 `payment.alipay.private_key_path` 和 `payment.alipay.alipay_public_key_path` 配置路径。
