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

RUN apt-get update && apt-get install -y ca-certificates curl

COPY ./cert/ca-certificates.crt /etc/ssl/certs/

RUN update-ca-certificates

# 设置工作目录
WORKDIR /server

# ---------- 安装 Chrome ----------
RUN apt-get update && apt-get install -y --no-install-recommends curl gnupg ca-certificates \
 && install -m 0755 -d /etc/apt/keyrings \
 && curl -fsSL https://dl.google.com/linux/linux_signing_key.pub | gpg --dearmor -o /etc/apt/keyrings/google-chrome.gpg \
 && chmod a+r /etc/apt/keyrings/google-chrome.gpg \
 && echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/google-chrome.gpg] http://dl.google.com/linux/chrome/deb/ stable main" \
      > /etc/apt/sources.list.d/google-chrome.list \
 && apt-get update && apt-get install -y --no-install-recommends \
      google-chrome-stable \
      fonts-noto-cjk \
      libx11-6 libx11-xcb1 libxcb1 libxcomposite1 libxcursor1 \
      libxdamage1 libxi6 libxtst6 libglib2.0-0 libgtk-3-0 libpango-1.0-0 \
      libcairo2 libdrm2 libgbm1 libasound2t64 \
 && rm -rf /var/lib/apt/lists/*

# 复制所有项目文件到容器中
COPY . /server/

# 给可执行文件增加执行权限
RUN chmod +x /server/LanMei

# Chrome 可执行路径
ENV CHROME_PATH=/usr/bin/google-chrome


# 暴露容器运行的端口
EXPOSE 8080

# 启动容器时运行的命令
CMD [ "/server/LanMei" ]
