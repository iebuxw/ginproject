#!/bin/sh
CERT_DIR="$(dirname "$0")/certs"
mkdir -p "$CERT_DIR"
if [ -f "$CERT_DIR/server.crt" ] && [ -f "$CERT_DIR/server.key" ]; then
    echo "证书已存在，跳过生成"
    exit 0
fi
umask 077
openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
  -keyout "$CERT_DIR/server.key" -out "$CERT_DIR/server.crt" \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1" || { echo '证书生成失败'; exit 1; }
echo "证书已生成: $CERT_DIR/server.crt"
