package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

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
	name     string
	id       int
	kind     FileKind
	dirPath  []int
	modTime  time.Time
	filePath string
}

type Directory struct {
	id             int
	name           string
	files          []File
	childDirectory []Directory
	modTime        time.Time
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
	log.Printf("addFileToContext: path=%s dirPath=%v", path, dirPath)
	cx.paths = append(cx.paths, path)
	fileInfo, err := os.Stat(path)
	if err != nil {
		return File{}, err
	}

	name := fileInfo.Name()
	id := len(cx.flatFiles)
	file := File{name: name, id: id, kind: fileKind(path), dirPath: dirPath, modTime: fileInfo.ModTime(), filePath: path}
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
	log.Printf("addDirRootToContext: path=%s currentPath=%v", path, currentPath)

	name := filepath.Base(path)
	// Capture directory metadata
	dirInfo, err := os.Stat(path)
	if err != nil {
		return Directory{}, err
	}
	directory := Directory{id: dirId, name: name, files: []File{}, childDirectory: []Directory{}, modTime: dirInfo.ModTime()}
	cx.directories = append(cx.directories, directory)

	childDirectory := make([]Directory, 0)
	files := make([]File, 0)

	entries, err := os.ReadDir(path)
	if err != nil {
		return Directory{}, err
	}

	for _, entry := range entries {
		childPath := filepath.Join(path, entry.Name())
		if filteredFile(childPath) {
			continue
		}
		if entry.IsDir() {
			// Recurse into child directories, but only add their metadata here
			dir, err := addDirRootToContext(cx, childPath, currentPath)
			if err != nil {
				return Directory{}, err
			}
			childDirectory = append(childDirectory, dir)
		} else {
			file, err := addFileToContext(cx, childPath, currentPath)
			if err != nil {
				return Directory{}, err
			}
			files = append(files, file)
		}
	}

	cx.directories[dirId].childDirectory = childDirectory
	cx.directories[dirId].files = files

	log.Printf("scanned directory: name=%s id=%d files=%d children=%d", name, dirId, len(files), len(childDirectory))

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

func directoryUrlWithSort(path, name, sortParam string) string {
	url := directoryUrl(path, name)
	if sortParam == "" {
		return url
	}
	return url + "?sort=" + sortParam
}

func childDirectoryData(directory *Directory, path string, sortParam string) []DirectoryData {
	// copy to avoid mutating original slice order
	dirs := make([]Directory, len(directory.childDirectory))
	copy(dirs, directory.childDirectory)
	// sort according to sortParam
	switch strings.ToLower(sortParam) {
	case "latest":
		sort.SliceStable(dirs, func(i, j int) bool { return dirs[i].modTime.After(dirs[j].modTime) })
	case "oldest":
		sort.SliceStable(dirs, func(i, j int) bool { return dirs[i].modTime.Before(dirs[j].modTime) })
	default: // name
		sort.SliceStable(dirs, func(i, j int) bool { return strings.ToLower(dirs[i].name) < strings.ToLower(dirs[j].name) })
	}

	data := make([]DirectoryData, 0, len(dirs))
	for _, dir := range dirs {
		data = append(data, DirectoryData{Name: dir.name, Url: directoryUrlWithSort(path, dir.name, sortParam)})
	}
	return data
}

func slideUrl(path string, file File, sortParam string) string {
	prefix := "/slides/"
	basePath := strings.Trim(path, "/")
	id := strconv.Itoa(file.id)
	var url string
	if basePath == "" {
		url = prefix + id
	} else {
		url = prefix + id + "/" + basePath
	}
	if sortParam != "" {
		url = url + "?sort=" + sortParam
	}
	return url
}

func fullscreenUrl(path string, file File, sortParam string) string {
	prefix := "/fullscreen/"
	basePath := strings.Trim(path, "/")
	id := strconv.Itoa(file.id)
	var url string
	if basePath == "" {
		url = prefix + id
	} else {
		url = prefix + id + "/" + basePath
	}
	if sortParam != "" {
		url = url + "?sort=" + sortParam
	}
	return url
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





func getSortedMediaFiles(files []File, sortParam string) []File {
	mediaFiles := make([]File, 0)
	for _, f := range files {
		if f.kind != Other {
			mediaFiles = append(mediaFiles, f)
		}
	}
	switch strings.ToLower(sortParam) {
	case "latest":
		sort.SliceStable(mediaFiles, func(i, j int) bool { return mediaFiles[i].modTime.After(mediaFiles[j].modTime) })
	case "oldest":
		sort.SliceStable(mediaFiles, func(i, j int) bool { return mediaFiles[i].modTime.Before(mediaFiles[j].modTime) })
	default: // name
		sort.SliceStable(mediaFiles, func(i, j int) bool { return strings.ToLower(mediaFiles[i].name) < strings.ToLower(mediaFiles[j].name) })
	}
	return mediaFiles
}

func fileDataInRange(cx *Context, directory *Directory, path string, start int, end int, sortParam string) []FileData {
	data := make([]FileData, 0)
	mediaFiles := getSortedMediaFiles(directory.files, sortParam)

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
			Url:          slideUrl(path, file, sortParam),
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
	// TODO: refactor image and video handlers
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
		sortParam := strings.ToLower(c.DefaultQuery("sort", "name"))
		returnImageGrid(&cx, c, directoryId, start, end, path, sortParam)
	})

	r.GET("/slides/:id/*path", func(c *gin.Context) {
		path := c.Param("path")
		sortParam := strings.ToLower(c.DefaultQuery("sort", "name"))
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
		sortedFiles := getSortedMediaFiles(directory.files, sortParam)
		index := index(sortedFiles, id)
		if index == -1 {
			handleInvalidFile(c, "File not found")
			return
		}
		isVideo := sortedFiles[index].kind == Video
		prev := prevUrl(sortedFiles, index, path, sortParam)
		next := nextUrl(sortedFiles, index, path, sortParam)
		resourceUrl := fileResourceUrl(sortedFiles[index])

		// Calculate centered grid position for current file (in sorted order)
		currentPosition := index
		totalFiles := len(sortedFiles)
		gridStart := currentPosition - PageSize/2
		gridEnd := currentPosition + PageSize/2

		// Handle negative wrap-around
		if gridStart < 0 && totalFiles > 0 {
			gridStart = totalFiles + gridStart
		}

		filePath := cx.buildFilePath(sortedFiles[index])
		c.HTML(http.StatusOK, "slide.tmpl", gin.H{
			"Name":        sortedFiles[index].name,
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
			"Sort":        sortParam,
		})
	})

	r.GET("/fullscreen/:id/*path", func(c *gin.Context) {
		path := c.Param("path")
		sortParam := strings.ToLower(c.DefaultQuery("sort", "name"))
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
		sortedFiles := getSortedMediaFiles(directory.files, sortParam)
		index := index(sortedFiles, id)
		if index == -1 {
			handleInvalidFile(c, "File not found")
			return
		}

		isVideo := sortedFiles[index].kind == Video
		prev := prevFullscreenUrl(sortedFiles, index, path, sortParam)
		next := nextFullscreenUrl(sortedFiles, index, path, sortParam)
		resourceUrl := fileResourceUrl(sortedFiles[index])

		// Count total images/videos for counter based on sorted files
		totalCount := len(sortedFiles)

		// Current position (1-based)
		currentIndex := index + 1

		c.HTML(http.StatusOK, "fullscreen.tmpl", gin.H{
			"Name":              sortedFiles[index].name,
			"IsVideo":           isVideo,
			"ResourceUrl":       resourceUrl,
			"PrevUrl":           prev,
			"NextUrl":           next,
			"CurrentIndex":      currentIndex,
			"TotalCount":        totalCount,
			"IsFirst":           index == 0,
			"IsLast":            index == len(sortedFiles)-1,
			"SlideshowInterval": SlideshowInterval,
			"Sort":              sortParam,
		})
	})

	// Delete endpoint: move file to system trash, update in-memory directory listing, redirect
	r.POST("/delete", func(c *gin.Context) {
		fileIdStr := c.PostForm("fileId")
		dirIdStr := c.PostForm("directoryId")
		pathParam := c.PostForm("path")
		sortParam := strings.ToLower(c.DefaultPostForm("sort", "name"))

		fileId, err := strconv.Atoi(fileIdStr)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}
		dirId, err := strconv.Atoi(dirIdStr)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}

		dir, err := cx.getDirectoryById(dirId)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}

		// Determine next slide target before deletion
		sortedFiles := getSortedMediaFiles(dir.files, sortParam)
		currIdx := index(sortedFiles, fileId)
		if currIdx == -1 {
			handleInvalidFile(c, "File not found in directory")
			return
		}

		nextTargetUrl := ""
		if len(sortedFiles) > 1 {
			nextTargetUrl = nextUrl(sortedFiles, currIdx, pathParam, sortParam)
		}

		// Trash the file
		pathToFile, err := cx.getPath(fileId)
		if err != nil {
			handleInvalidFile(c, err.Error())
			return
		}

		if err := moveToTrash(pathToFile); err != nil {
			// If file already doesn't exist, proceed as if deleted
			if _, statErr := os.Stat(pathToFile); statErr == nil {
				handleError(c, http.StatusInternalServerError, fmt.Sprintf("failed to move to trash: %v", err))
				return
			}
		}

		// Update in-memory directory listing: remove the file with matching id
		filtered := make([]File, 0, len(dir.files))
		for _, f := range dir.files {
			if f.id != fileId {
				filtered = append(filtered, f)
			}
		}
		dir.files = filtered

		// If there are remaining files, go to next slide; otherwise back to directory view
		if len(filtered) > 0 && nextTargetUrl != "" {
			c.Redirect(http.StatusSeeOther, nextTargetUrl)
			return
		}

		// Redirect back to directory page
		redir := "/files" + pathParam
		if sortParam != "" {
			redir = redir + "?sort=" + sortParam
		}
		c.Redirect(http.StatusSeeOther, redir)
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

func nextUrl(files []File, i int, path string, sortParam string) string {
	if i+1 < len(files) {
		return slideUrl(path, files[i+1], sortParam)
	} else {
		return slideUrl(path, files[0], sortParam)
	}
}

func prevUrl(files []File, i int, path string, sortParam string) string {
	if i-1 >= 0 {
		return slideUrl(path, files[i-1], sortParam)
	} else {
		return slideUrl(path, files[len(files)-1], sortParam)
	}
}

func nextFullscreenUrl(files []File, i int, path string, sortParam string) string {
	if i+1 < len(files) {
		return fullscreenUrl(path, files[i+1], sortParam)
	} else {
		return fullscreenUrl(path, files[0], sortParam)
	}
}

func prevFullscreenUrl(files []File, i int, path string, sortParam string) string {
	if i-1 >= 0 {
		return fullscreenUrl(path, files[i-1], sortParam)
	} else {
		return fullscreenUrl(path, files[len(files)-1], sortParam)
	}
}

func returnDirectoryPage(c *gin.Context, cx *Context, directory *Directory, path string) {
	sortParam := strings.ToLower(c.DefaultQuery("sort", "name"))
	Directories := childDirectoryData(directory, path, sortParam)
	hasDirectories := len(Directories) > 0
	hasFiles := countMediaFiles(directory) > 0
	c.HTML(http.StatusOK, "directoryData.tmpl", gin.H{
		"name":           directory.name,
		"Directories":    Directories,
		"DirectoryId":    directory.id,
		"Path":           path,
		"Sort":           sortParam,
		"HasDirectories": hasDirectories,
		"HasFiles":       hasFiles,
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

func returnImageGrid(cx *Context, c *gin.Context, directoryId int, start int, end int, path string, sortParam string) {
	directory, err := cx.getDirectoryById(directoryId)
	if err != nil {
		handleInvalidFile(c, err.Error())
		return
	}
	files := fileDataInRange(cx, directory, path, start, end, sortParam)
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
		"Sort":        sortParam,
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

// moveToTrash attempts to move the file to the system's recycle bin.
// On Linux, it tries common mechanisms in order.
func moveToTrash(path string) error {
	// Prefer gio (GLib) trash which adheres to the FreeDesktop Trash spec
	if err := tryExec("gio", "trash", path); err == nil {
		return nil
	}
	// Older gvfs-trash
	if err := tryExec("gvfs-trash", path); err == nil {
		return nil
	}
	// trash-put from trash-cli
	if err := tryExec("trash-put", path); err == nil {
		return nil
	}
	// KDE kioclient5
	if err := tryExec("kioclient5", "move", path, "trash:/"); err == nil {
		return nil
	}
	return fmt.Errorf("no trash utility found (tried gio, gvfs-trash, trash-put, kioclient5)")
}

func tryExec(name string, args ...string) error {
	if _, err := exec.LookPath(name); err != nil {
		return err
	}
	cmd := exec.Command(name, args...)
	return cmd.Run()
}
