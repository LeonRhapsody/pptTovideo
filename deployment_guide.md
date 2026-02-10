# 服务器部署指南 (推荐 Docker 方案)

由于本项目依赖多个系统级组件（FFmpeg, LibreOffice, Poppler），直接在服务器上安装这些包可能比较繁琐。推荐使用 **Docker** 进行一键部署，这样可以确保环境完全一致。

## 1. 准备工作

确保您的服务器已安装：
- Docker
- Docker Compose

## 2. 部署步骤

### 方法 A：使用 Docker Compose (推荐)

1. **克隆代码** 后，进入项目根目录。
2. **启动容器**：
   ```bash
   docker-compose up -d
   ```
3. **查看日志**：
   ```bash
   docker-compose logs -f
   ```

### 方法 B：手动部署 (如果您不想用 Docker)

您需要手动安装以下包（以 Ubuntu 为例）：

```bash
sudo apt-get update
sudo apt-get install -y ffmpeg libreoffice poppler-utils fonts-noto-cjk
```

然后编译运行：
```bash
go build -o main cmd/main.go
./main
```

## 3. 注意事项

- **端口**：默认运行在 `8080` 端口。如果需要修改，请在 `Dockerfile` 或 `docker-compose.yml` 中调整。
- **持久化**：`uploads/` 目录和 `config.json` 文件已在 `docker-compose.yml` 中配置了挂载，确保重启容器后数据和配置不会丢失。
- **内存建议**：LibreOffice 在解析大型 PPT 时可能会占用较多内存，建议服务器配置不少于 2GB 内存。

## 4. 常见问题

- **中文字体**：Docker 镜像中已包含 `fonts-noto-cjk`，可以完美支持视频中的中文字幕显示。
- **权限问题**：如果在生成过程中遇到权限错误，请确保 `uploads/` 目录对 Docker 运行用户有写入权限。
