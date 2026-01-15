#!/bin/bash
# Setup EC2 Ubuntu cho JoblyBE
# Chạy: chmod +x setup-ec2.sh && sudo ./setup-ec2.sh

# Kiểm tra quyền root
if [ "$EUID" -ne 0 ]; then
  echo "Vui lòng chạy với sudo: sudo ./setup-ec2.sh"
  exit 1
fi

echo "=== Cài đặt Docker & Docker Compose cho EC2 ==="

# Update system
echo "[1/4] Update system..."
sudo apt-get update -y

# Cài Docker
echo "[2/4] Cài Docker..."
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
rm get-docker.sh

# Thêm user vào docker group
sudo usermod -aG docker ubuntu

# Cài Docker Compose
echo "[3/4] Cài Docker Compose..."
sudo apt-get install -y docker-compose-plugin

# Cài Git
echo "[4/4] Cài Git..."
sudo apt-get install -y git

# Verify
echo ""
echo "=== Cài đặt hoàn tất! ==="
docker --version
docker compose version
git --version

echo ""
echo ">>> QUAN TRỌNG: Logout và login lại để dùng Docker không cần sudo <<<"
echo ""
echo "Sau khi login lại, chạy:"
echo "  cd ~"
echo "  git clone YOUR_REPO_URL jobly"
echo "  cd jobly"
echo "  docker compose up -d --build"

