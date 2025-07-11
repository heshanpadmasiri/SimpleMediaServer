# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SimpleMediaServer is a Go-based web application that serves images and videos from specified directories through a web interface. It provides:
- Directory browsing with thumbnails
- Slideshow functionality for images and videos  
- Fullscreen viewing mode
- Lazy loading with HTMX
- Support for multiple media source directories

## Architecture

### Core Components

**main.go** - Contains the entire application logic:
- HTTP server setup using Gin framework
- File system scanning and indexing 
- Template rendering for web UI
- Route handlers for different views (directory, slides, fullscreen, etc.)

**config.go** - Configuration management:
- TOML-based configuration loading
- Support for single or multiple media sources
- Automatic path validation

**Templates** (`templates/`):
- HTML templates using Go template syntax
- HTMX integration for dynamic loading
- Responsive grid layouts for media files

**Static Assets** (`static/`):
- CSS styling for the web interface

### Data Model

The application uses in-memory data structures:
- `Context` - Global state containing file paths and directory tree
- `Directory` - Represents a filesystem directory with files and subdirectories  
- `File` - Represents individual media files with metadata (type, ID, name)

### Key Design Patterns

1. **File Indexing**: All files are scanned at startup and assigned unique integer IDs for URL routing
2. **Virtual Directory**: When multiple media sources are configured, they're combined under a virtual "Collections" root
3. **Type-based Routing**: Separate URL patterns for images (`/img/:id`) and videos (`/video/:id`)
4. **Lazy Loading**: Directory contents loaded via HTMX requests to `/image-grid/` endpoints

## Development Commands

### Building and Running
```bash
# Build the application
go build -o media-server .

# Run directly with Go
go run .

# Run the built binary
./media-server
```

### Configuration
- Copy `config.toml.example` to `config.toml` and edit media paths
- Alternative location: `~/.config/simpleMediaServer/config.toml`
- Multiple media sources supported via `MediaSources` array
- Default port is 8080 if not specified

### Dependencies
```bash
# Download dependencies
go mod download

# Update dependencies  
go mod tidy
```

## File Organization

- **Single-file architecture**: Core logic in `main.go` (587 lines)
- **Template-driven UI**: HTML templates in `templates/` directory
- **Static assets**: CSS and other static files in `static/`
- **Test data**: Sample media files in `testData/` for development
- **Configuration**: TOML-based config with example file provided

## Supported Media Types

**Images**: .jpg, .jpeg, .png, .gif, .webp, .bmp, .svg
**Videos**: .mp4, .webm, .ogg, .ogv, .mov

Files starting with '.' are automatically filtered out during scanning.

## Key Features to Understand

1. **Lazy Loading**: Uses HTMX to load image grids on demand
2. **Slideshow Navigation**: Circular navigation through media files  
3. **Fullscreen Mode**: Dedicated fullscreen viewing with counters
4. **Multiple Media Sources**: Can serve from multiple directory trees
5. **Range Request Support**: Proper HTTP range handling for video streaming

# Style guide
- Use Go's standard formatting (`gofmt`).
- Where possible use HTMX for interactivity over javascript.
- Where possible use standard html elements and go standard library.
- Don't add comments describing each code block, instead extract blocks into functions with descriptive names.
