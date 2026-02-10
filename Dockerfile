FROM ubuntu:latest

# 设置国内的 apt 镜像源
RUN sed -i 's|http://archive.ubuntu.com/ubuntu/|http://mirrors.aliyun.com/ubuntu/|g' /etc/apt/sources.list
RUN apt-get update && apt-get install -y \
    libc6 \
    libgcc1 \
    tzdata \
    ntpdate \
    && rm -rf /var/lib/apt/lists/*

# 设置时区
RUN echo "Asia/Shanghai" > /etc/timezone && dpkg-reconfigure -f noninteractive tzdata

RUN ntpdate -s time.nist.gov

RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && dpkg-reconfigure -f noninteractive tzdata

# 设置工作目录
WORKDIR /server

# 复制所有项目文件到容器中
COPY . /server/

# 给可执行文件增加执行权限
RUN chmod +x /server/lanmei

# Chrome 可执行路径
ENV CHROME_PATH=/usr/bin/google-chrome


# 暴露容器运行的端口
EXPOSE 8080

# 启动容器时运行的命令
CMD [ "/server/lanmei" ]
