# 🎬 Media Server - Gin/Go

A lightweight media file server written in Go using the Gin web framework.  
Supports browsing, uploading, downloading media files, and generating thumbnails via `ffmpeg`.

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

- [ ] Add logging middleware or structured logs.

- [ ] Add unit tests for handlers.

- [ ] Add JWT-based authentication.

- [ ] Pagination support for /media.

- [ ] Delete media API.

- [ ] Config file support (e.g., media root path, port).

- [ ] Use SQLite or other persistent DB instead of in-memory map.

- [ ] API documentation (Swagger or Postman collection).

- [ ] Dockerize the app.
