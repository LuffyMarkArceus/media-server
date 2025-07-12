# 🎬 Media Server - Gin/Go

A lightweight media file server written in Go using the Gin web framework.  
Supports browsing, uploading, downloading media files, and generating thumbnails & subtitles (if any) via `ffmpeg`.

---

## 🚀 Getting Started

### 1. Install Go dependencies

```bash
go mod init media-server
go mod tidy
```

### 2. Install ffmpeg (for thumbnail generation)

```bash
# On Ubuntu/Debian
sudo apt install ffmpeg

# On Mac
brew install ffmpeg

# On Windows, download ffmpeg from https://ffmpeg.org/download.html
```

### 3. Run the server

```bash
go run main.go
```

#### For dockerized builds:

```bash
docker build -t <your_dockerhub_username>/media-server:latest .
docker run -p 8080:8080 <your_dockerhub_username>/media-server:latest

# docker logs <container_id>
# docker stop <container_id>
```

### Backend Tasks - Gin Go Server

- [x] Set up Gin project.
- [x] File upload via /upload.
- [x] List all media files under /media.
- [x] Serve files via /media/\*filepath.
- [x] Generate thumbnails using ffmpeg.
- [x] Support nested folder structure in /thumbnail/\*filepath.
- [x] Modify DB Schema for media files & folders
- [x] Config file support, .env (e.g., media root path, port).
- [x] Use SQLite or other persistent DB instead of in-memory map.
- [x] Rename media route.

- [ ] Delete media route.
- [x] Upload media route.

- [ ] Add logging middleware or structured logs.
- [ ] Add unit tests for handlers.
- [ ] Add JWT-based authentication.
- [ ] Pagination support for /media.
- [ ] API documentation (Swagger or Postman collection).
- [x] Dockerize the app.
- [x] Deploy to Google Cloud Run.

### BONUS Ideas

- [ ] Return video duration from ffprobe or ffmpeg during upload/scan.
- [x] Add /health endpoint for monitoring.
