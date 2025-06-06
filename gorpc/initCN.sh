#!/bin/bash

# --- 脚本配置 ---
PROJECT_ROOT="$HOME/GoRPC"

# Etcd 容器名称
ETCD_CONTAINER_NAME="etcd-server"
# Etcd 镜像和版本
ETCD_IMAGE="bitnami/etcd:3.5.0"
# Etcd 宿主机端口（客户端端口）
ETCD_HOST_PORT="2389"
# Etcd 容器内部客户端端口
ETCD_CONTAINER_CLIENT_PORT="2379"
# Etcd 宿主机对等端口
ETCD_HOST_PEER_PORT="2380"
# Etcd 容器内部对等端口
ETCD_CONTAINER_PEER_PORT="2380"

# RPC 服务器和客户端的路径
SERVER_PATH="server/main.go"
CLIENT_PATH="client/main.go"

# 日志文件
ETCD_LOG="/tmp/etcd_start.log"
SERVER_LOG="/tmp/rpc_server.log"
CLIENT_LOG="/tmp/rpc_client.log"

# --- 脚本开始 ---

echo "--- 正在启动 RPC 框架与 Etcd 服务栈 ---"
echo "项目根目录: $PROJECT_ROOT"

# 1. 停止并移除旧的 Etcd 容器（如果存在）
echo -e "\n--- 1. 停止并移除旧的 Etcd 容器 (${ETCD_CONTAINER_NAME}) ---"
if sudo docker ps -a --format '{{.Names}}' | grep -q "${ETCD_CONTAINER_NAME}"; then
    echo "发现旧容器，正在停止并移除..."
    sudo docker stop ${ETCD_CONTAINER_NAME} > /dev/null 2>&1
    sudo docker rm ${ETCD_CONTAINER_NAME} > /dev/null 2>&1
    echo "旧容器已移除。"
else
    echo "未发现旧的 Etcd 容器，跳过移除。"
fi

# 2. 启动 Etcd Docker 容器
echo -e "\n--- 2. 启动 Etcd Docker 容器 (${ETCD_IMAGE}) ---"
echo "Etcd 将映射到主机端口 ${ETCD_HOST_PORT}:${ETCD_CONTAINER_CLIENT_PORT} 和 ${ETCD_HOST_PEER_PORT}:${ETCD_CONTAINER_PEER_PORT}"
sudo docker run -d \
    -p ${ETCD_HOST_PORT}:${ETCD_CONTAINER_CLIENT_PORT} \
    -p ${ETCD_HOST_PEER_PORT}:${ETCD_CONTAINER_PEER_PORT} \
    --name ${ETCD_CONTAINER_NAME} \
    -e ALLOW_NONE_AUTHENTICATION=yes \
    ${ETCD_IMAGE} > ${ETCD_LOG} 2>&1

# 检查 Etcd 是否成功启动
sleep 3 # 等待 Etcd 容器启动
ETCD_STATUS=$(sudo docker ps -a --format '{{.Status}}' --filter "name=${ETCD_CONTAINER_NAME}")
if [[ "$ETCD_STATUS" == *"Up"* ]]; then
    echo "Etcd 容器已成功启动并运行。日志文件: ${ETCD_LOG}"
else
    echo "错误：Etcd 容器启动失败！请检查日志文件: ${ETCD_LOG}"
    exit 1
fi

# 3. 启动 RPC 服务器
echo -e "\n--- 3. 启动 RPC 服务器 (Go) ---"
echo "RPC 服务器日志文件: ${SERVER_LOG}"
cd "${PROJECT_ROOT}" || { echo "错误: 无法进入项目目录 ${PROJECT_ROOT}"; exit 1; }
go run "${SERVER_PATH}" > "${SERVER_LOG}" 2>&1 &
SERVER_PID=$! # 获取服务器进程ID

# 等待服务器启动并注册服务
echo "等待 RPC 服务器启动并注册到 Etcd..."
sleep 5 # 给予服务器足够的时间启动和注册

# 检查服务器日志确认Etcd注册成功
if grep -q "Service registered in Etcd:" "${SERVER_LOG}"; then
    echo "RPC 服务器已成功启动并注册到 Etcd。"
else
    echo "错误：RPC 服务器启动失败或未成功注册到 Etcd！请检查日志文件: ${SERVER_LOG}"
    kill $SERVER_PID # 杀死服务器进程
    exit 1
fi

# 4. 启动 RPC 客户端
echo -e "\n--- 4. 启动 RPC 客户端 (Go) ---"
echo "RPC 客户端日志文件: ${CLIENT_LOG}"
# 客户端直接运行，不放后台，因为它会自行退出
go run "${CLIENT_PATH}" > "${CLIENT_LOG}" 2>&1 

# 客户端运行结束后，等待一会儿
echo "RPC 客户端运行完成。请查看日志文件: ${CLIENT_LOG}"
sleep 1

# 5. 提醒用户如何停止服务
echo -e "\n--- 5. 服务栈已全部启动/运行完成 ---"
echo "您现在可以查看日志文件以了解详细输出："
echo "  Etcd 容器启动日志: ${ETCD_LOG}"
echo "  RPC 服务器日志: ${SERVER_LOG}"
echo "  RPC 客户端日志: ${CLIENT_LOG}"
echo -e "\n若要停止 RPC 服务器，请运行：kill ${SERVER_PID}"
echo "若要停止并移除 Etcd 容器，请运行：sudo docker stop ${ETCD_CONTAINER_NAME} && sudo docker rm ${ETCD_CONTAINER_NAME}"
echo "或者要停止所有后台进程：killall go" # 简单粗暴，不推荐用于生产
echo "--- 脚本结束 ---"