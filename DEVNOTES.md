# 二开 / 部署踩坑记录 (DEVNOTES)

> 记录 sublinkX 二开与 VPS 部署过程中踩过的坑，避免下次重复犯错。

## 环境
- 上游仓库: https://github.com/gooaclok819/sublinkX
- 二开仓库: https://github.com/Xioaruan912/SubL
- VPS: HostDZire `199.47.242.40` (root, ssh 22)
- 生产目录: `/usr/local/bin/sublink/`（含 db/ template/ logs/ 与可执行文件 `sublink`）
- 生产服务: `sublink.service`（systemd，监听 8001，nginx:80 反代）

## 踩坑记录

### 1. 大文件 scp 上传超时（25MB 二进制）
- 现象：`scp` 单个 25MB 二进制直接超时失败（`shell tool terminated after 120s`）。
- 解决：**分片上传**。本地 `split -b 2m -d file part_`，逐片 scp，VPS 上 `cat part_* > new.bin`，最后 `md5sum` 校验一致。
- 教训：跨低带宽链路传大文件务必分片 + 校验 md5。

### 2. VPS 上已有生产 sublinkX 在跑，不能盲目覆盖
- 现象：`/usr/local/bin/sublink` 目录已有 active 的 `sublink.service`，监听 8001，带真实用户数据（db/sublink.db、config.yaml 含 jwt_secret）。
- 解决：先 `systemctl stop sublink`，**完整备份** `/usr/local/bin/sublink/{db,template,logs}` 和旧二进制到 `/root/backup/sublink_<ts>/`，再替换二进制、重启、验证。
- 教训：**部署前必须探查目标主机现有服务**（`systemctl list-units`, `ss -tlnp`, 目录内容），先备份再操作。

### 3. 新二进制是否兼容旧生产数据？先本地冒烟测试
- 现象：无法确定 main 分支新编译二进制能否直接用旧 sqlite 数据库启动。
- 解决：把备份的 `db/` 拉到本地，用新二进制 `run -port 19999` 启动冒烟，确认 `数据库初始化成功`（AutoMigrate 无报错）、页面 200、API 正常、db 文件大小不变。
- 教训：**数据库/配置升级前，用备份数据在本地先跑通**，确认 schema 兼容再上生产。
- 备注：新编译二进制 md5 `dd177088...`；旧生产二进制 md5 与上游 release 2.1 完全一致 `46bf3962...`，即生产本来就是 2.1。

### 4. 配置与数据目录位置（工作目录相关）
- sublinkX 是**工作目录敏感**程序：`./db/config.yaml`、`./db/sublink.db`、`./template/`、`./logs/` 都在二进制所在工作目录下。
- systemd `ExecStart=/usr/local/bin/sublink/sublink` + `WorkingDirectory=/usr/local/bin/sublink`，所以替换二进制时**只动顶层 `sublink` 文件，绝不能动 db/template/logs**。
- 换端口不要改代码，直接改 `db/config.yaml` 里的 `port`。

### 5. 登录/API 路由
- 登录路由是 `POST /api/v1/auth/login`，需先 `GET /api/v1/auth/captcha` 拿验证码，直接 POST 会返回「请求未携带token」。
- 首页 `/` 返回 200 即为前端正常。

## 部署速查（原地升级）
```bash
# 1) 停止 + 备份
systemctl stop sublink
TS=$(date +%Y%m%d_%H%M%S); mkdir -p /root/backup/sublink_$TS
cp -a /usr/local/bin/sublink/{db,template,logs} /root/backup/sublink_$TS/
cp /usr/local/bin/sublink/sublink /root/backup/sublink_$TS/sublink.old.bin

# 2) 分片上传新二进制（本地）
split -b 2m -d sublink_amd64 /tmp/sublink_part_
# scp 每片到 VPS /tmp 后：
cat /tmp/sublink_part_* > /tmp/sublink.new.bin; md5sum /tmp/sublink.new.bin

# 3) 替换并重启
mv /tmp/sublink.new.bin /usr/local/bin/sublink/sublink; chmod +x ...
systemctl start sublink
systemctl status sublink; ss -tlnp | grep 8001

# 4) 回滚（如需）
systemctl stop sublink
cp /root/backup/sublink_<ts>/sublink.old.bin /usr/local/bin/sublink/sublink
systemctl start sublink
```
