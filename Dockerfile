# 使用 Ubuntu 镜像作为基础镜像
FROM ubuntu:latest

# 设置国内的 apt 镜像源
RUN sed -i 's|http://archive.ubuntu.com/ubuntu/|http://mirrors.aliyun.com/ubuntu/|g' /etc/apt/sources.list
RUN apt-get update && apt-get install -y \
    libc6 \
    libgcc1 \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

RUN apt-get update && apt-get install -y ca-certificates curl

COPY ./cert/ca-certificates.crt /etc/ssl/certs/

RUN update-ca-certificates

# 设置工作目录
WORKDIR /server

# 复制所有项目文件到容器中
COPY . /server/

# 给可执行文件增加执行权限
RUN chmod +x /server/LanMei

# 暴露容器运行的端口
EXPOSE 8080

# 启动容器时运行的命令
CMD [ "/server/LanMei" ]
