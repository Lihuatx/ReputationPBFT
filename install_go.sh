#!/bin/bash

# 设置Go语言版本
GO_VERSION="1.20.1"
GO_TAR_FILE="go$GO_VERSION.linux-amd64.tar.gz"

# 下载地址
DOWNLOAD_URL="https://go.dev/dl/$GO_TAR_FILE"

# 安装路径
INSTALL_DIR="/usr/local"

# 删除旧的Go安装
if [ -d "$INSTALL_DIR/go" ]; then
    echo "Removing previous Go installation."
    sudo rm -rf $INSTALL_DIR/go
fi

# 下载Go语言压缩包
echo "Downloading Go $GO_VERSION..."
wget $DOWNLOAD_URL -O $GO_TAR_FILE

# 解压到安装目录
echo "Extracting Go $GO_VERSION to $INSTALL_DIR"
sudo tar -C $INSTALL_DIR -xzf $GO_TAR_FILE

# 清理下载的文件
rm $GO_TAR_FILE

# 设置Go环境变量
echo "Setting up Go environment variables"
echo "export PATH=\$PATH:$INSTALL_DIR/go/bin" >> ~/.profile
echo "export GOROOT=$INSTALL_DIR/go" >> ~/.profile
echo "export GOPATH=\$HOME/go" >> ~/.profile
echo "export GOBIN=\$GOPATH/bin" >> ~/.profile

# 重新加载profile以立即生效
source ~/.profile

echo "Go $GO_VERSION installation completed."
go version
