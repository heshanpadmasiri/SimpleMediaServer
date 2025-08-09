package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type FileKind int

const (
	Image FileKind = iota
	Video
	Other
)

const (
	PageSize          = 10
	SlideshowInterval = 5000 // 5 seconds in milliseconds
)

type File struct {
	name    string
	id      int
	kind    FileKind
	dirPath []int
}

type Directory struct {
	id             int
	name           string
	files          []File
	childDirectory []Directory
}

type Context struct {
	paths       []string
	directories []Directory
	flatFiles   []File
}

// given file index get the path from context
func (cx *Context) getPath(id int) (string, error) {
	if id < 0 || id >= len(cx.paths) {
		return "", fmt.Errorf("invalid id %d", id)
	}
	return cx.paths[id], nil
}

func (cx *Context) getNextDirectoryId() int {
	return len(cx.directories)
}

func (cx *Context) getDirectoryById(id int) (*Directory, error) {
	if id < 0 || id >= len(cx.directories) {
		return nil, fmt.Errorf("invalid directory id %d", id)
	}
	return &cx.directories[id], nil
}

// buildFilePath constructs the full path to a file using its dirPath array with clickable links
func (cx *Context) buildFilePath(file File) string {
	if len(file.dirPath) == 0 {
		return file.name
	}

	pathParts := make([]string, 0, len(file.dirPath)+1)
	currentPath := ""

	// Build path parts with links for directories
	for _, dirId := range file.dirPath {
		if dirId == 0 {
			// Virtual root di
			continue
		}
		if dirId >= 0 && dirId < len(cx.directories) {
			dirName := cx.directories[dirId].name
			if currentPath == "" {
				currentPath = dirName
			} else {
				currentPath = currentPath + "/" + dirName
			}
			link := fmt.Sprintf(`<a href="/files/%s">%s</a>`, currentPath, dirName)
			pathParts = append(pathParts, link)
		}
	}

	// Add the file name (not clickable)
	pathParts = append(pathParts, file.name)

	return strings.Join(pathParts, " / ")
}

func splitPath(path string) (string, string) {
	split := strings.SplitN(path, "/", 2)
	if len(split) == 1 {
		return split[0], ""
	}
	return split[0], split[1]
}

func getDirectoryByPath(root *Directory, path string) *Directory {
	path = strings.Trim(path, "/")
	if path == "" {
		return root
	}
	dirName, rest := splitPath(path)
	for _, dir := range root.childDirectory {
		if dir.name == dirName {
			return getDirectoryByPath(&dir, rest)
		}
	}
	return nil
}

func addFileToContext(cx *Context, path string, dirPath []int) (File, error) {
	fmt.Println(path, dirPath)
	cx.paths = append(cx.paths, path)
	fileInfo, err := os.Stat(path)
	if err != nil {
		return File{}, err
	}

	name := fileInfo.Name()
	id := len(cx.flatFiles)
	file := File{name: name, id: id, kind: fileKind(path), dirPath: dirPath}
	cx.flatFiles = append(cx.flatFiles, file)
	return file, nil
}

func fileKind(path string) FileKind {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return Image
	case ".mp4", ".webm", ".ogg", ".ogv", ".mov":
		return Video
	default:
		return Other
	}
}

func filteredFile(path string) bool {
	return filepath.Base(path)[0] == '.'
}

func addDirRootToContext(cx *Context, path string, parentPath []int) (Directory, error) {
	dirId := len(cx.directories)
	currentPath := append(parentPath, dirId)
	fmt.Println(path, currentPath)

	name := filepath.Base(path)
	directory := Directory{id: dirId, name: name, files: []File{}, childDirectory: []Directory{}}
	cx.directories = append(cx.directories, directory)

	childDirectory := make([]Directory, 0)
	files := make([]File, 0)
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if filePath == path || filteredFile(filePath) {
			return nil
		}
		if err != nil {
			return err
		}

		if info.IsDir() {
			dir, err := addDirRootToContext(cx, filePath, currentPath)
			if err != nil {
				return err
			}
			childDirectory = append(childDirectory, dir)
		} else {
			file, err := addFileToContext(cx, filePath, currentPath)
			if err != nil {
				return err
			}
			files = append(files, file)
		}

		return nil
	})

	if err != nil {
		return Directory{}, err
	}
	cx.directories[dirId].childDirectory = childDirectory
	cx.directories[dirId].files = files

	return cx.directories[dirId], nil
}

type DirectoryData struct {
	Name string
	Url  string
}

type FileData struct {
	Name         string
	Url          string
	ResourceUrl  string
	ThumbnailUrl string
	IsVideo      bool
}

func directoryUrl(path, name string) string {
	basePath := strings.Trim(path, "/")
	prefix := "/files/"
	if basePath == "" {
		return prefix + name
	}
	return prefix + basePath + "/" + name
}

func childDirectoryData(directory *Directory, path string) []DirectoryData {
	data := make([]DirectoryData, 0)
	for _, dir := range directory.childDirectory {
		data = append(data, DirectoryData{Name: dir.name, Url: directoryUrl(path, dir.name)})
	}
	return data
}

func slideUrl(path string, file File) string {
	prefix := "/slides/"
	basePath := strings.Trim(path, "/")
	id := strconv.Itoa(file.id)
	if basePath == "" {
		return prefix + id
	}
	return prefix + id + "/" + basePath
}

func fullscreenUrl(path string, file File) string {
	prefix := "/fullscreen/"
	basePath := strings.Trim(path, "/")
	id := strconv.Itoa(file.id)
	if basePath == "" {
		return prefix + id
	}
	return prefix + id + "/" + basePath
}

func fileResourceUrl(file File) string {
	switch file.kind {
	case Video:
		return videoResourceUrlById(file.id)
	case Image:
		return imageResourceUrlById(file.id)
	default:
		panic("unimplemented")
	}
}

func fileThumbnailUrl(cx *Context, file File) string {
	switch file.kind {
	case Video:
		// For videos, use the video file itself as thumbnail
		// The browser will display the first frame
		// FIXME: for this to work we need to use a video tag not an img tag
		return videoResourceUrlById(file.id)
	case Image:
		return imageResourceUrlById(file.id)
	default:
		panic("unimplemented")
	}
}

func videoResourceUrlById(id int) string {
	return "/video/" + strconv.Itoa(id)
}

func imageResourceUrlById(id int) string {
	return "/img/" + strconv.Itoa(id)
}

func getFilesInRange(cx *Context, directory *Directory, path string, index int) []FileData {
	files := directory.files
	if len(files) <= 11 {
		return getFilesInRangeInner(cx, path, files)
	}
	start, end := getIndexRange(index)
	return getFilesInRangeInner(cx, path, files[start:end])
}

func getFilesInRangeInner(cx *Context, path string, files []File) []FileData {
	data := make([]FileData, 0)
	for _, file := range files {
		if file.kind == Other {
			continue
		}
		data = append(data, FileData{
			Name:         file.name,
			Url:          slideUrl(path, file),
			ResourceUrl:  fileResourceUrl(file),
			ThumbnailUrl: fileThumbnailUrl(cx, file),
			IsVideo:      file.kind == Video,
		})
	}
	return data
}

func fileDataInner(cx *Context, directory *Directory, path string, limit int) []FileData {
	data := make([]FileData, 0)
	for _, file := range directory.files {
		if file.kind == Other {
			continue
		}
		url := slideUrl(path, file)
		if file.kind == Video {
			url = fileResourceUrl(file)
		}
		data = append(data, FileData{
			Name:         file.name,
			Url:          url,
			ResourceUrl:  fileResourceUrl(file),
			ThumbnailUrl: fileThumbnailUrl(cx, file),
			IsVideo:      file.kind == Video,
		})
		if len(data) == limit {
			break
		}
	}
	return data
}

func fileDataInRange(cx *Context, directory *Directory, path string, start int, end int) []FileData {
	data := make([]FileData, 0)
	mediaFiles := make([]File, 0)

	// First collect all media files
	for _, file := range directory.files {
		if file.kind != Other {
			mediaFiles = append(mediaFiles, file)
		}
	}

	totalFiles := len(mediaFiles)
	if totalFiles == 0 {
		return data
	}

	// Handle wrap-around by normalizing indices (including negative ones)
	for i := start; i < end; i++ {
		index := ((i % totalFiles) + totalFiles) % totalFiles
		file := mediaFiles[index]

		data = append(data, FileData{
			Name:         file.name,
			Url:          slideUrl(path, file),
			ResourceUrl:  fileResourceUrl(file),
			ThumbnailUrl: fileThumbnailUrl(cx, file),
			IsVideo:      file.kind == Video,
		})
	}

	return data
}

func countMediaFiles(directory *Directory) int {
	count := 0
	for _, file := range directory.files {
		if file.kind != Other {
			count++
		}
	}
	return count
}

func findMediaFilePosition(directory *Directory, fileId int) int {
	position := 0
	for _, file := range directory.files {
		if file.kind == Other {
			continue
		}
		if file.id == fileId {
			return position
		}
		position++
	}
	return -1
}

func main() {
	cx := Context{paths: make([]string, 0), directories: make([]Directory, 0), flatFiles: make([]File, 0)}

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	var rootDir Directory

	if len(config.MediaSources) == 1 {
		// Single media source - use it directly
		dir, err := addDirRootToContext(&cx, config.MediaSources[0], []int{})
		if err != nil {
			panic(err)
		}
		rootDir = dir
	} else {
		// Multiple media sources - create virtual directory
		directories := make([]Directory, 0, len(config.MediaSources))
		virtualDir := Directory{
			id:             0,
			name:           "$Virtual",
			files:          []File{},
			childDirectory: directories,
		}
		cx.directories = append(cx.directories, virtualDir)
		for _, path := range config.MediaSources {
			dir, err := addDirRootToContext(&cx, path, []int{0})
			if err != nil {
				log.Printf("Warning: failed to add directory %s: %v", path, err)
				continue
			}
			directories = append(directories, dir)
		}

		if len(directories) == 0 {
			log.Fatal("No valid media sources found")
		}
		rootDir = combineDirectories(&cx, directories)
	}
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		returnDirectoryPage(c, &cx, &rootDir, "")
	})
	r.GET("/files/*path", func(c *gin.Context) {
		path := c.Param("path")
		directory := getDirectoryByPath(&rootDir, path)
		if directory == nil {
			c.HTML(http.StatusNotFound, "invalidPath.tmpl", gin.H{
				"path": path,
			})
			return
		}
		returnDirectoryPage(c, &cx, directory, path)
	})
	// TODO: refactor image and vidoe handlers
	r.GET("/img/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}
		returnFileById(&cx, c, id)
	})
	r.GET("/video/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}
		returnFileById(&cx, c, id)
	})
	r.GET("/image-grid/:directoryId/*path", func(c *gin.Context) {
		directoryIdStr := c.Param("directoryId")
		directoryId, err := strconv.Atoi(directoryIdStr)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}

		startStr := c.DefaultQuery("start", "0")
		start, err := strconv.Atoi(startStr)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}

		endStr := c.DefaultQuery("end", "10")
		end, err := strconv.Atoi(endStr)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}

		path := c.Param("path")
		returnImageGrid(&cx, c, directoryId, start, end, path)
	})

	r.GET("/slides/:id/*path", func(c *gin.Context) {
		path := c.Param("path")
		idStr := c.Param("id")
		directory := getDirectoryByPath(&rootDir, path)
		if directory == nil {
			c.HTML(http.StatusNotFound, "invalidPath.tmpl", gin.H{
				"path": path,
			})
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}
		fmt.Println(cx.flatFiles[id])
		index := index(directory.files, id)
		if index == -1 {
			handleInvalidFile(c, "File not found")
			return
		}
		isVideo := directory.files[index].kind == Video
		prev := prevUrl(directory.files, index, path)
		next := nextUrl(directory.files, index, path)
		resourceUrl := fileResourceUrl(directory.files[index])

		// Calculate centered grid position for current file
		currentPosition := findMediaFilePosition(directory, id)
		totalFiles := countMediaFiles(directory)
		gridStart := currentPosition - PageSize/2
		gridEnd := currentPosition + PageSize/2

		// Handle negative wrap-around
		if gridStart < 0 && totalFiles > 0 {
			gridStart = totalFiles + gridStart
		}

		filePath := cx.buildFilePath(directory.files[index])
		c.HTML(http.StatusOK, "slide.tmpl", gin.H{
			"Name":        directory.files[index].name,
			"FilePath":    template.HTML(filePath),
			"isVideo":     isVideo,
			"ResourceUrl": resourceUrl,
			"PrevUrl":     prev,
			"NextUrl":     next,
			"DirectoryId": directory.id,
			"FileId":      id,
			"Path":        path,
			"GridStart":   gridStart,
			"GridEnd":     gridEnd,
		})
	})

	r.GET("/fullscreen/:id/*path", func(c *gin.Context) {
		path := c.Param("path")
		idStr := c.Param("id")
		directory := getDirectoryByPath(&rootDir, path)
		if directory == nil {
			c.HTML(http.StatusNotFound, "invalidPath.tmpl", gin.H{
				"path": path,
			})
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}
		index := index(directory.files, id)
		if index == -1 {
			handleInvalidFile(c, "File not found")
			return
		}

		isVideo := directory.files[index].kind == Video
		prev := prevFullscreenUrl(directory.files, index, path)
		next := nextFullscreenUrl(directory.files, index, path)
		resourceUrl := fileResourceUrl(directory.files[index])

		// Count total images/videos for counter
		totalCount := 0
		for _, file := range directory.files {
			if file.kind == Image || file.kind == Video {
				totalCount++
			}
		}

		// Count current position (only images/videos)
		currentIndex := 1
		for i := 0; i < index; i++ {
			if directory.files[i].kind == Image || directory.files[i].kind == Video {
				currentIndex++
			}
		}

		c.HTML(http.StatusOK, "fullscreen.tmpl", gin.H{
			"Name":              directory.files[index].name,
			"IsVideo":           isVideo,
			"ResourceUrl":       resourceUrl,
			"PrevUrl":           prev,
			"NextUrl":           next,
			"CurrentIndex":      currentIndex,
			"TotalCount":        totalCount,
			"IsFirst":           index == 0,
			"IsLast":            index == len(directory.files)-1,
			"SlideshowInterval": SlideshowInterval,
		})
	})

	r.Run(fmt.Sprintf(":%d", config.Port))
}

func getIndexRange(index int) (int, int) {
	start := index - 5
	if start < 0 {
		start = 0
	}
	end := start + 10
	return start, end
}

func index(files []File, id int) int {
	for i, file := range files {
		if file.id == id {
			return i
		}
	}
	return -1
}

func nextUrl(files []File, i int, path string) string {
	if i+1 < len(files) {
		return slideUrl(path, files[i+1])
	} else {
		log.Println(files[0])
		return slideUrl(path, files[0])
	}
}

func prevUrl(files []File, i int, path string) string {
	if i-1 >= 0 {
		return slideUrl(path, files[i-1])
	} else {
		return slideUrl(path, files[len(files)-1])
	}
}

func nextFullscreenUrl(files []File, i int, path string) string {
	if i+1 < len(files) {
		return fullscreenUrl(path, files[i+1])
	} else {
		return fullscreenUrl(path, files[0])
	}
}

func prevFullscreenUrl(files []File, i int, path string) string {
	if i-1 >= 0 {
		return fullscreenUrl(path, files[i-1])
	} else {
		return fullscreenUrl(path, files[len(files)-1])
	}
}

func returnDirectoryPage(c *gin.Context, cx *Context, directory *Directory, path string) {
	Directories := childDirectoryData(directory, path)
	c.HTML(http.StatusOK, "directoryData.tmpl", gin.H{
		"name":        directory.name,
		"Directories": Directories,
		"DirectoryId": directory.id,
		"Path":        path,
	})
}

func returnFileByPath(c *gin.Context, path string) {
	file, err := os.Open(path)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		handleError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Use http.ServeContent for better HTTP serving with range requests support
	http.ServeContent(c.Writer, c.Request, fileInfo.Name(), fileInfo.ModTime(), file)
}

func returnFileById(cx *Context, c *gin.Context, id int) {
	path, err := cx.getPath(id)
	if err != nil {
		handleInvalidFile(c, err.Error())
		return
	}
	returnFileByPath(c, path)
}

func returnImageGrid(cx *Context, c *gin.Context, directoryId int, start int, end int, path string) {
	directory, err := cx.getDirectoryById(directoryId)
	if err != nil {
		handleInvalidFile(c, err.Error())
		return
	}
	files := fileDataInRange(cx, directory, path, start, end)
	nextStart := end
	nextEnd := end + PageSize
	totalFiles := countMediaFiles(directory)

	// Always show more button since we have wrap-around
	// Only hide if there are no files at all
	hasMore := totalFiles > 0

	c.HTML(http.StatusOK, "imageGrid.tmpl", gin.H{
		"Files":       files,
		"HasMore":     hasMore,
		"NextStart":   nextStart,
		"NextEnd":     nextEnd,
		"DirectoryId": directoryId,
		"Path":        path,
	})
}

func handleError(c *gin.Context, statusCode int, reason string) {
	log.Printf("Error: %s", reason)
	c.HTML(statusCode, "error.tmpl", gin.H{
		"reason": reason,
	})
}

func handleInvalidFile(c *gin.Context, reason string) {
	log.Printf("Invalid file error: %s", reason)
	c.HTML(http.StatusNotFound, "invalidFile.tmpl", gin.H{
		"reason": reason,
	})
}

// combineDirectories creates a virtual directory that contains all the provided directories as child directories
func combineDirectories(cx *Context, directories []Directory) Directory {
	// Create a virtual root directory
	virtualDir := Directory{
		id:             len(cx.directories),
		name:           "Collections",
		files:          []File{},
		childDirectory: directories,
	}

	// Add the virtual directory to the context
	cx.directories = append(cx.directories, virtualDir)

	return virtualDir
}
