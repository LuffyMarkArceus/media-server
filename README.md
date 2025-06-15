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

- [ ] Delete media route.
- [x] Rename media route.
- [ ] Upload media route.

- [ ] Add logging middleware or structured logs.
- [ ] Add unit tests for handlers.
- [ ] Add JWT-based authentication.
- [ ] Pagination support for /media.
- [ ] API documentation (Swagger or Postman collection).
- [ ] Dockerize the app.

### BONUS Ideas

- [ ] Return video duration from ffprobe or ffmpeg during upload/scan.
- [ ]
- [ ] Use Range header support with c.FileFromFS() or http.ServeContent for efficient streaming.
- [-] Add /healthz and /metrics endpoints for monitoring.
