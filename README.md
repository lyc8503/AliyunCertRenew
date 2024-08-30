# AliyunCertRenew

本程序用于自动续期阿里云云上资源(如 CDN/函数计算)的免费 SSL 证书, 可无需服务器部署

## 背景

[之前](https://help.aliyun.com/zh/ssl-certificate/product-overview/notice-on-adjustment-of-service-policies-for-free-certificates)阿里云将个人免费证书的时长调整为了三个月.  
云上资源也无法使用 ACME 自动申请部署新证书, 每三个月就需要手动登陆控制台续期, 且只会在证书过期前一个月有短信提醒.  
操作繁琐之外, 如果忘记续期还会导致意外的服务中断.

## 准备工作

请先确认以下内容:
1. 你的域名 DNS 解析已由阿里云托管 (用以完成域名 DNS 认证)
2. 目前你已在阿里云上申请并部署 SSL 证书 (程序会自动按照当前的配置续期)
3. 创建一个 [RAM 子用户](https://ram.console.aliyun.com/users), 授予 `AliyunYundunCertFullAccess` 权限, 创建并保存一对 AccessKey ID 和 AccessKey Secret

## 部署

程序从环境变量读取所需参数, 必填参数如下:

1. `DOMAIN`: 需要证书续期的域名, 多个域名用英文逗号分隔
2. `ACCESS_KEY_ID`: 上面提到的子用户 AccessKey ID
3. `ACCESS_KEY_SECRET`: 上面提到的子用户 AccessKey Secret

可选参数有 `DEBUG`, 当指定为任意非空值时会输出详细的调试日志

程序会检查对应 `DOMAIN` 中所有的域名, 如果存在域名的证书过期时间在 7 天内, 则会申请新的免费证书, 部署替换当前证书.

### 1. 使用 GitHub Actions 部署(无需服务器)

Fork 本仓库后, 在你自己的仓库中, 进入 `Settings - Secrets and variables - Actions`, 添加上面必填的三个环境变量作为 `Repository secrets`.

添加完成后应该类似下图

![](https://github.com/user-attachments/assets/ec7bc16a-f66c-47ce-9f23-7cfb24b9a8e1)

切换到 `Actions` 标签页, 点击左侧 `Check Certificate` 任务, 手动 `Run workflow` 一次, 如成功则代表配置正确, 配置的 cron 会自动运行后续的续期任务.

### 2. 本地部署运行

从 Releases 中下载对应本地架构的二进制文件, 直接运行即可. 可以使用 crontab 定期运行, 建议每三天执行一次.

## 效果

一次正常的续期运行日志如下, 续期和部署成功后, 阿里云均会给用户发送短信和邮件提醒:

```
AliyunCertRenew version 497b999
INFO[0000] AliyunCertRenew starting...                  
INFO[0000] Domains to check: [example.com]             
INFO[0000] >>> Checking example.com                    
INFO[0000] Certificate renewal needed for example.com  
INFO[0000] New certificate request created for example.com 
INFO[0010] Order current status: CHECKING               
INFO[0020] Order current status: CHECKING               
INFO[0031] Order current status: CHECKING               
INFO[0041] Order current status: CHECKING               
INFO[0051] Order current status: CHECKING               
INFO[0061] Order current status: CHECKING               
INFO[0071] Order current status: CHECKING               
INFO[0085] Order current status: ISSUED                 
INFO[0085] New cert created for example.com: 14764653  
INFO[0090] Deployment job created: 92612
INFO[0094] Deployment job submitted: 92612
```