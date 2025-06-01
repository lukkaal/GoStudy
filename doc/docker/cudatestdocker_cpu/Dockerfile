FROM ubuntu:22.04

# 安装 Python 和系统工具
RUN apt-get update && \
    apt-get install -y python3 python3-pip git wget && \
    rm -rf /var/lib/apt/lists/*

# 设置工作目录
WORKDIR /workspace

# 安装深度学习库
COPY requirements.txt .
RUN pip3 install --no-cache-dir -r requirements.txt

# 复制示例代码（可选）
COPY example.py .

# 默认进入 bash（适合交互）
CMD ["/bin/bash"]
