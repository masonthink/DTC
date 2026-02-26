#!/usr/bin/env bash
# VPS 初始化脚本（Debian 12 / Ubuntu 22.04）
# 在全新 Hetzner VPS 上以 root 执行一次：
#   bash <(curl -fsSL https://raw.githubusercontent.com/mason2047/DTC/main/deploy/setup.sh)
#
# 完成后 deploy 用户可通过 SSH 部署，root 登录将被禁用。

set -euo pipefail

DEPLOY_USER="deploy"
APP_DIR="/opt/digital-twin"

echo "==> 更新系统"
apt-get update -qq && apt-get upgrade -y -qq

echo "==> 安装 Docker"
curl -fsSL https://get.docker.com | sh
systemctl enable --now docker

echo "==> 创建部署用户 ${DEPLOY_USER}"
id -u "${DEPLOY_USER}" &>/dev/null || useradd -m -s /bin/bash "${DEPLOY_USER}"
usermod -aG docker "${DEPLOY_USER}"

echo "==> 配置 SSH 公钥（粘贴 GitHub Actions 部署密钥的公钥）"
mkdir -p /home/${DEPLOY_USER}/.ssh
chmod 700 /home/${DEPLOY_USER}/.ssh
# 将 GitHub Actions 部署密钥公钥写入此文件
touch /home/${DEPLOY_USER}/.ssh/authorized_keys
chmod 600 /home/${DEPLOY_USER}/.ssh/authorized_keys
chown -R ${DEPLOY_USER}:${DEPLOY_USER} /home/${DEPLOY_USER}/.ssh

echo "==> 创建应用目录"
mkdir -p "${APP_DIR}"
chown "${DEPLOY_USER}:${DEPLOY_USER}" "${APP_DIR}"

echo "==> 配置防火墙（UFW）"
apt-get install -y -qq ufw
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP（Caddy 重定向）
ufw allow 443/tcp   # HTTPS
ufw allow 443/udp   # HTTP/3
ufw --force enable

echo "==> 禁用 root SSH 登录"
sed -i 's/^PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config
sed -i 's/^#PermitRootLogin/PermitRootLogin no/' /etc/ssh/sshd_config
systemctl reload sshd

echo ""
echo "✓ 初始化完成。接下来："
echo "  1. 将 GitHub Actions 部署公钥写入 /home/${DEPLOY_USER}/.ssh/authorized_keys"
echo "  2. 将 docker-compose.prod.yml、deploy/ 复制到 ${APP_DIR}"
echo "  3. 在 ${APP_DIR} 创建 .env.prod 文件并填入生产配置"
echo "  4. 以 ${DEPLOY_USER} 用户执行首次启动："
echo "     docker compose -f docker-compose.prod.yml up -d"
