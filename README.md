# SimpleMediaServer

A lightweight Go-based web application that serves images and videos from local directories through a responsive web interface. Features directory browsing, slideshow functionality, fullscreen viewing, and lazy loading.

## Features

- **Directory Browsing**: Navigate through media directories with thumbnail previews
- **Slideshow Mode**: Sequential viewing of images and videos with navigation
- **Fullscreen Viewing**: Dedicated fullscreen mode with media counters
- **Multiple Media Sources**: Support for serving content from multiple directory trees
- **Lazy Loading**: HTMX-powered dynamic content loading for better performance
- **Video Streaming**: HTTP range request support for efficient video playback

### Supported Media Types

- **Images**: `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`, `.bmp`, `.svg`
- **Videos**: `.mp4`, `.webm`, `.ogg`, `.ogv`, `.mov`

## Quick Start

### Native Installation

1. **Clone and build:**
   ```bash
   git clone https://github.com/heshanpadmasiri/SimpleMediaServer.git
   cd SimpleMediaServer
   go build -o media-server .
   ```

2. **Configure media sources:**
   ```bash
   cp config.toml.example config.toml
   # Edit config.toml to specify your media directories
   ```

3. **Run the server:**
   ```bash
   ./media-server
   ```

Access the web interface at `http://localhost:8080`

### Docker

1. **Using Docker Compose (Recommended):**
   ```bash
   # Edit docker-compose.yml to mount your media directories
   docker-compose up -d
   ```

2. **Using Docker directly:**
   ```bash
   docker build -t simple-media-server .
   docker run -p 8080:8080 -v /path/to/your/media:/media:ro simple-media-server
   ```

## Configuration

### Configuration File Format

The application uses TOML format for configuration. The config file supports the following options:

```toml
# Port to run the server on (optional, default: 8080)
Port = 8080

# Multiple media sources (recommended)
MediaSources = [
    "/path/to/your/photos",
    "/path/to/your/videos",
    "/another/media/directory"
]

# Single media source (legacy, ignored if MediaSources is provided)
# MediaSource = "/single/path/to/media"
```

### Configuration File Locations

The application searches for configuration files in the following order:

1. **Current directory**: `./config.toml`
2. **User config directory**: `~/.config/simpleMediaServer/config.toml`

If neither file is found, the application will exit with an error message.

### Example Configurations

**Multiple media sources:**
```toml
Port = 8080
MediaSources = [
    "/Users/username/Pictures",
    "/Users/username/Movies",
    "/shared/photos"
]
```

**Windows paths:**
```toml
MediaSources = [
    "C:\\Users\\username\\Pictures",
    "D:\\Media\\Videos"
]
```

**Single source (legacy):**
```toml
MediaSource = "/home/user/media"
```

## Docker Deployment

### Building the Container

The container is built using a multi-stage Dockerfile that produces a lightweight Alpine-based image.

```bash
docker build -t simple-media-server .
```

### Media Directory Mounting

When running in Docker, the container expects media to be mounted at `/media`. The container includes a pre-configured `config.container.toml` that points to this location.

**Docker Compose example:**
```yaml
version: '3.8'
services:
  media-server:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ~/Media/:/media:ro  # Mount your media directory
    restart: unless-stopped
```

**Docker run example:**
```bash
# Mount single directory
docker run -p 8080:8080 -v /path/to/media:/media:ro simple-media-server

# Mount multiple directories using bind mounts
docker run -p 8080:8080 \
  -v /path/to/photos:/media/photos:ro \
  -v /path/to/videos:/media/videos:ro \
  simple-media-server
```

### Container Security

The Docker container runs with security hardening:
- Non-root user (`mediaserver:1001`)
- Read-only filesystem
- No new privileges
- Minimal Alpine base image

## Development

### Building from Source

```bash
# Install dependencies
go mod download

# Build the application
go build -o media-server .

# Run directly with Go
go run .
```

### Development Dependencies

- Go 1.22.6 or later
- Dependencies are managed with Go modules

### Project Structure

```
SimpleMediaServer/
├── main.go              # Core application logic
├── config.go            # Configuration management
├── templates/           # HTML templates
├── static/             # CSS and static assets
├── testData/           # Sample media files
├── config.toml.example # Example configuration
├── Dockerfile          # Container build instructions
├── docker-compose.yml  # Docker Compose configuration
└── config.container.toml # Container-specific config
```

## API Endpoints

- `/` - Main directory view
- `/dir/:path` - Browse specific directory
- `/img/:id` - Serve image by ID
- `/video/:id` - Serve video by ID
- `/slides/:path` - Slideshow view
- `/fullscreen/:id` - Fullscreen media view
- `/image-grid/:path` - HTMX lazy-loaded grid content

## License

This project is licensed under the MIT License. See the LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

For bug reports and feature requests, please use the GitHub issue tracker.