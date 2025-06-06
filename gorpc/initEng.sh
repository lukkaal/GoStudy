#!/bin/bash

# --- Script Configuration ---
PROJECT_ROOT="$HOME/GoRPC"

# Etcd container name
ETCD_CONTAINER_NAME="etcd-server"
# Etcd image and version
ETCD_IMAGE="bitnami/etcd:3.5.0"
# Etcd host port (client port)
ETCD_HOST_PORT="2389"
# Etcd container internal client port
ETCD_CONTAINER_CLIENT_PORT="2379"
# Etcd host peer port
ETCD_HOST_PEER_PORT="2380"
# Etcd container internal peer port
ETCD_CONTAINER_PEER_PORT="2380"

# Paths to RPC server and client
SERVER_PATH="server/main.go"
CLIENT_PATH="client/main.go"

# Log files
ETCD_LOG="/tmp/etcd_start.log"
SERVER_LOG="/tmp/rpc_server.log"
CLIENT_LOG="/tmp/rpc_client.log"

# --- Script Start ---

echo "--- Starting RPC Framework and Etcd Service Stack ---"
echo "Project Root: $PROJECT_ROOT"

# 1. Stop and remove old Etcd container (if it exists)
echo -e "\n--- 1. Stopping and Removing Old Etcd Container (${ETCD_CONTAINER_NAME}) ---"
if sudo docker ps -a --format '{{.Names}}' | grep -q "${ETCD_CONTAINER_NAME}"; then
    echo "Found old container, stopping and removing..."
    sudo docker stop ${ETCD_CONTAINER_NAME} > /dev/null 2>&1
    sudo docker rm ${ETCD_CONTAINER_NAME} > /dev/null 2>&1
    echo "Old container removed."
else
    echo "No old Etcd container found, skipping removal."
fi

# 2. Start Etcd Docker container
echo -e "\n--- 2. Starting Etcd Docker Container (${ETCD_IMAGE}) ---"
echo "Etcd will be mapped to host ports ${ETCD_HOST_PORT}:${ETCD_CONTAINER_CLIENT_PORT} and ${ETCD_HOST_PEER_PORT}:${ETCD_CONTAINER_PEER_PORT}"
sudo docker run -d \
    -p ${ETCD_HOST_PORT}:${ETCD_CONTAINER_CLIENT_PORT} \
    -p ${ETCD_HOST_PEER_PORT}:${ETCD_CONTAINER_PEER_PORT} \
    --name ${ETCD_CONTAINER_NAME} \
    -e ALLOW_NONE_AUTHENTICATION=yes \
    ${ETCD_IMAGE} > ${ETCD_LOG} 2>&1

# Check if Etcd started successfully
sleep 3 # Wait for the Etcd container to start
ETCD_STATUS=$(sudo docker ps -a --format '{{.Status}}' --filter "name=${ETCD_CONTAINER_NAME}")
if [[ "$ETCD_STATUS" == *"Up"* ]]; then
    echo "Etcd container started and is running successfully. Log file: ${ETCD_LOG}"
else
    echo "Error: Etcd container failed to start! Please check the log file: ${ETCD_LOG}"
    exit 1
fi

# 3. Start RPC Server
echo -e "\n--- 3. Starting RPC Server (Go) ---"
echo "RPC Server log file: ${SERVER_LOG}"
cd "${PROJECT_ROOT}" || { echo "Error: Cannot enter project directory ${PROJECT_ROOT}"; exit 1; }
go run "${SERVER_PATH}" > "${SERVER_LOG}" 2>&1 &
SERVER_PID=$! # Get the server process ID

# Wait for the server to start and register the service
echo "Waiting for RPC server to start and register with Etcd..."
sleep 5 # Give the server enough time to start and register

# Check server logs to confirm Etcd registration success
if grep -q "Service registered in Etcd:" "${SERVER_LOG}"; then
    echo "RPC Server started and registered with Etcd successfully."
else
    echo "Error: RPC Server failed to start or did not register with Etcd! Please check the log file: ${SERVER_LOG}"
    kill $SERVER_PID # Kill the server process
    exit 1
fi

# 4. Start RPC Client
echo -e "\n--- 4. Starting RPC Client (Go) ---"
echo "RPC Client log file: ${CLIENT_LOG}"
# The client runs directly and exits on its own
go run "${CLIENT_PATH}" > "${CLIENT_LOG}" 2>&1 

# After the client finishes, wait a bit
echo "RPC Client has finished running. Please check the log file: ${CLIENT_LOG}"
sleep 1

# 5. Instructions on how to stop services
echo -e "\n--- 5. Service Stack All Launched/Completed ---"
echo "You can now view the log files for detailed output:"
echo "  Etcd Container Startup Log: ${ETCD_LOG}"
echo "  RPC Server Log: ${SERVER_LOG}"
echo "  RPC Client Log: ${CLIENT_LOG}"
echo -e "\nTo stop the RPC server, run: kill ${SERVER_PID}"
echo "To stop and remove the Etcd container, run: sudo docker stop ${ETCD_CONTAINER_NAME} && sudo docker rm ${ETCD_CONTAINER_NAME}"
echo "Alternatively, to stop all background Go processes: killall go" # Simple but not recommended for production
echo "--- Script End ---"