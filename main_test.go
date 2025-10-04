package main

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test Helper Functions

func createTestFile(id int, name string, kind FileKind, modTime time.Time, deleted bool) File {
	return File{
		id:       id,
		name:     name,
		kind:     kind,
		modTime:  modTime,
		deleted:  deleted,
		dirPath:  []int{},
		filePath: "/test/" + name,
	}
}

func createTestDirectory(id int, name string, fileIds []int, modTime time.Time) Directory {
	return Directory{
		id:             id,
		name:           name,
		files:          fileIds,
		childDirectory: []Directory{},
		modTime:        modTime,
	}
}

func createTestContext(files []File, directories []Directory) *Context {
	return &Context{
		files:       files,
		directories: directories,
		rootDir:     nil,
	}
}

// validateURL checks if a string is a valid URL path
func validateURL(urlStr string) bool {
	_, err := url.ParseRequestURI(urlStr)
	return err == nil
}

// Test fileKind function
func TestFileKind_ImageExtensions_ReturnsImage(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"JPEG", "photo.jpg"},
		{"JPEG uppercase", "PHOTO.JPG"},
		{"JPEG alternate", "image.jpeg"},
		{"PNG", "image.png"},
		{"GIF", "animation.gif"},
		{"WebP", "modern.webp"},
		{"BMP", "bitmap.bmp"},
		{"SVG", "vector.svg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fileKind(tt.path)
			assert.Equal(t, Image, result)
		})
	}
}

func TestFileKind_VideoExtensions_ReturnsVideo(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"MP4", "video.mp4"},
		{"MP4 uppercase", "VIDEO.MP4"},
		{"WebM", "video.webm"},
		{"OGG", "video.ogg"},
		{"OGV", "video.ogv"},
		{"MOV", "video.mov"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fileKind(tt.path)
			assert.Equal(t, Video, result)
		})
	}
}

func TestFileKind_UnknownExtension_ReturnsOther(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"PDF", "document.pdf"},
		{"TXT", "readme.txt"},
		{"No extension", "filename"},
		{"Multiple dots", "file.tar.gz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fileKind(tt.path)
			assert.Equal(t, Other, result)
		})
	}
}

// Test filteredFile function
func TestFilteredFile_HiddenFile_ReturnsTrue(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"Hidden file", ".hidden"},
		{"Hidden with extension", ".gitignore"},
		{"Hidden in path", "/path/to/.hidden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filteredFile(tt.path)
			assert.True(t, result)
		})
	}
}

func TestFilteredFile_RegularFile_ReturnsFalse(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"Regular file", "file.txt"},
		{"File with dot in name", "my.file.txt"},
		{"Path with hidden parent", "/path/.hidden/visible.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filteredFile(tt.path)
			assert.False(t, result)
		})
	}
}

// Test splitPath function
func TestSplitPath_SingleSegment_ReturnsSingleElementAndEmpty(t *testing.T) {
	first, rest := splitPath("folder")
	assert.Equal(t, "folder", first)
	assert.Equal(t, "", rest)
}

func TestSplitPath_MultipleSegments_SplitsOnFirstSeparator(t *testing.T) {
	first, rest := splitPath("folder/subfolder/file")
	assert.Equal(t, "folder", first)
	assert.Equal(t, "subfolder/file", rest)
}

func TestSplitPath_EmptyString_ReturnsEmptyAndEmpty(t *testing.T) {
	first, rest := splitPath("")
	assert.Equal(t, "", first)
	assert.Equal(t, "", rest)
}

func TestSplitPath_LeadingSeparator_ReturnsEmptyFirstSegment(t *testing.T) {
	first, rest := splitPath("/folder/file")
	assert.Equal(t, "", first)
	assert.Equal(t, "folder/file", rest)
}

func TestSplitPath_TrailingSeparator_PreservesIt(t *testing.T) {
	first, rest := splitPath("folder/")
	assert.Equal(t, "folder", first)
	assert.Equal(t, "", rest)
}

// Test sortByFromStr function
func TestSortByFromStr_ValidValues_ReturnsCorrectSortBy(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected SortBy
	}{
		{"Latest lowercase", "latest", Latest},
		{"Latest uppercase", "LATEST", Latest},
		{"Latest mixed", "Latest", Latest},
		{"Oldest lowercase", "oldest", Oldest},
		{"Oldest uppercase", "OLDEST", Oldest},
		{"Name lowercase", "name", Name},
		{"Name uppercase", "NAME", Name},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sortByFromStr(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortByFromStr_InvalidValue_ReturnsError(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Invalid string", "invalid"},
		{"Empty string", ""},
		{"Random string", "random"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sortByFromStr(tt.input)
			assert.Error(t, err)
		})
	}
}

// Test SortBy.toStr method
func TestSortByToStr_ValidValues_ReturnsCorrectString(t *testing.T) {
	tests := []struct {
		name     string
		sortBy   SortBy
		expected string
	}{
		{"Latest", Latest, "latest"},
		{"Oldest", Oldest, "oldest"},
		{"Name", Name, "name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.sortBy.toStr()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortByToStr_InvalidValue_ReturnsDefault(t *testing.T) {
	invalidSortBy := SortBy(-1)
	result := invalidSortBy.toStr()
	assert.Equal(t, "name", result)
}

// Test directoryUrl function
func TestDirectoryUrl_EmptyPath_ReturnsCorrectUrl(t *testing.T) {
	result := directoryUrl("", "folder")
	assert.Equal(t, "/files/folder", result)
	assert.True(t, validateURL(result))
}

func TestDirectoryUrl_NonEmptyPath_ReturnsCorrectUrl(t *testing.T) {
	result := directoryUrl("parent", "child")
	assert.Equal(t, "/files/parent/child", result)
	assert.True(t, validateURL(result))
}

func TestDirectoryUrl_PathWithLeadingSlash_TrimsSlash(t *testing.T) {
	result := directoryUrl("/parent", "child")
	assert.Equal(t, "/files/parent/child", result)
	assert.True(t, validateURL(result))
}

func TestDirectoryUrl_PathWithTrailingSlash_TrimsSlash(t *testing.T) {
	result := directoryUrl("parent/", "child")
	assert.Equal(t, "/files/parent/child", result)
	assert.True(t, validateURL(result))
}

func TestDirectoryUrl_EmptyName_Panics(t *testing.T) {
	assert.Panics(t, func() {
		directoryUrl("path", "")
	})
}

// Test directoryUrlWithSort function
func TestDirectoryUrlWithSort_EmptySortParam_ReturnsUrlWithoutSort(t *testing.T) {
	result := directoryUrlWithSort("path", "folder", "")
	assert.Equal(t, "/files/path/folder", result)
	assert.True(t, validateURL(result))
}

func TestDirectoryUrlWithSort_ValidSortParam_ReturnsUrlWithSort(t *testing.T) {
	result := directoryUrlWithSort("path", "folder", "latest")
	assert.Equal(t, "/files/path/folder?sort=latest", result)
	assert.True(t, validateURL(result))
}

func TestDirectoryUrlWithSort_EmptyName_Panics(t *testing.T) {
	assert.Panics(t, func() {
		directoryUrlWithSort("path", "", "latest")
	})
}

// Test slideUrl function
func TestSlideUrl_EmptyPath_ReturnsCorrectUrl(t *testing.T) {
	file := createTestFile(42, "test.jpg", Image, time.Now(), false)
	result := slideUrl("", file, "")
	assert.Equal(t, "/slides/42", result)
	assert.True(t, validateURL(result))
}

func TestSlideUrl_NonEmptyPath_ReturnsCorrectUrl(t *testing.T) {
	file := createTestFile(42, "test.jpg", Image, time.Now(), false)
	result := slideUrl("parent/child", file, "")
	assert.Equal(t, "/slides/42/parent/child", result)
	assert.True(t, validateURL(result))
}

func TestSlideUrl_WithSortParam_AppendsSortParam(t *testing.T) {
	file := createTestFile(42, "test.jpg", Image, time.Now(), false)
	result := slideUrl("path", file, "latest")
	assert.Equal(t, "/slides/42/path?sort=latest", result)
	assert.True(t, validateURL(result))
}

// Test fullscreenUrl function
func TestFullscreenUrl_EmptyPath_ReturnsCorrectUrl(t *testing.T) {
	file := createTestFile(42, "test.jpg", Image, time.Now(), false)
	result := fullscreenUrl("", file, "")
	assert.Equal(t, "/fullscreen/42", result)
	assert.True(t, validateURL(result))
}

func TestFullscreenUrl_NonEmptyPath_ReturnsCorrectUrl(t *testing.T) {
	file := createTestFile(42, "test.jpg", Image, time.Now(), false)
	result := fullscreenUrl("parent/child", file, "")
	assert.Equal(t, "/fullscreen/42/parent/child", result)
	assert.True(t, validateURL(result))
}

func TestFullscreenUrl_WithSortParam_AppendsSortParam(t *testing.T) {
	file := createTestFile(42, "test.jpg", Image, time.Now(), false)
	result := fullscreenUrl("path", file, "latest")
	assert.Equal(t, "/fullscreen/42/path?sort=latest", result)
	assert.True(t, validateURL(result))
}

// Test fileResourceURL function
func TestFileResourceURL_ImageFile_ReturnsImageUrl(t *testing.T) {
	modTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	file := createTestFile(42, "test.jpg", Image, modTime, false)
	result := fileResourceURL(file)
	assert.Contains(t, result, "/img/42")
	assert.Contains(t, result, "?t=")
	assert.True(t, validateURL(result))
}

func TestFileResourceURL_VideoFile_ReturnsVideoUrl(t *testing.T) {
	modTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	file := createTestFile(42, "test.mp4", Video, modTime, false)
	result := fileResourceURL(file)
	assert.Contains(t, result, "/video/42")
	assert.Contains(t, result, "?t=")
	assert.True(t, validateURL(result))
}

func TestFileResourceURL_DifferentModTimes_ProducesDifferentUrls(t *testing.T) {
	modTime1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	modTime2 := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)
	file1 := createTestFile(42, "test.jpg", Image, modTime1, false)
	file2 := createTestFile(42, "test.jpg", Image, modTime2, false)

	result1 := fileResourceURL(file1)
	result2 := fileResourceURL(file2)
	assert.NotEqual(t, result1, result2)
}

// Test fileThumbnailURL function
func TestFileThumbnailURL_ReturnsSameAsFileResourceURL(t *testing.T) {
	modTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	file := createTestFile(42, "test.jpg", Image, modTime, false)

	resourceURL := fileResourceURL(file)
	thumbnailURL := fileThumbnailURL(file)

	assert.Equal(t, resourceURL, thumbnailURL)
}

// Test index function
func TestIndex_FileExists_ReturnsCorrectIndex(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
		createTestFile(2, "c.jpg", Image, time.Now(), false),
	}

	tests := []struct {
		name     string
		id       int
		expected int
	}{
		{"First file", 0, 0},
		{"Middle file", 1, 1},
		{"Last file", 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := index(files, tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIndex_FileDoesNotExist_ReturnsMinusOne(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
	}

	result := index(files, 99)
	assert.Equal(t, -1, result)
}

func TestIndex_EmptyArray_ReturnsMinusOne(t *testing.T) {
	files := []File{}
	result := index(files, 0)
	assert.Equal(t, -1, result)
}

// Test nextUrl function
func TestNextUrl_MiddleElement_ReturnsNext(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
		createTestFile(2, "c.jpg", Image, time.Now(), false),
	}

	result := nextUrl(files, 1, "path", "")
	assert.Equal(t, "/slides/2/path", result)
}

func TestNextUrl_LastElement_WrapsToFirst(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
		createTestFile(2, "c.jpg", Image, time.Now(), false),
	}

	result := nextUrl(files, 2, "path", "")
	assert.Equal(t, "/slides/0/path", result)
}

func TestNextUrl_SingleElement_ReturnsItself(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
	}

	result := nextUrl(files, 0, "path", "")
	assert.Equal(t, "/slides/0/path", result)
}

func TestNextUrl_WithSortParam_IncludesSortInUrl(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
	}

	result := nextUrl(files, 0, "path", "latest")
	assert.Equal(t, "/slides/1/path?sort=latest", result)
}

// Test prevUrl function
func TestPrevUrl_MiddleElement_ReturnsPrevious(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
		createTestFile(2, "c.jpg", Image, time.Now(), false),
	}

	result := prevUrl(files, 1, "path", "")
	assert.Equal(t, "/slides/0/path", result)
}

func TestPrevUrl_FirstElement_WrapsToLast(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
		createTestFile(2, "c.jpg", Image, time.Now(), false),
	}

	result := prevUrl(files, 0, "path", "")
	assert.Equal(t, "/slides/2/path", result)
}

func TestPrevUrl_SingleElement_ReturnsItself(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
	}

	result := prevUrl(files, 0, "path", "")
	assert.Equal(t, "/slides/0/path", result)
}

func TestPrevUrl_WithSortParam_IncludesSortInUrl(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
	}

	result := prevUrl(files, 1, "path", "latest")
	assert.Equal(t, "/slides/0/path?sort=latest", result)
}

// Test nextFullscreenUrl function
func TestNextFullscreenUrl_MiddleElement_ReturnsNext(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
		createTestFile(2, "c.jpg", Image, time.Now(), false),
	}

	result := nextFullscreenUrl(files, 1, "path", "")
	assert.Equal(t, "/fullscreen/2/path", result)
}

func TestNextFullscreenUrl_LastElement_WrapsToFirst(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
		createTestFile(2, "c.jpg", Image, time.Now(), false),
	}

	result := nextFullscreenUrl(files, 2, "path", "")
	assert.Equal(t, "/fullscreen/0/path", result)
}

// Test prevFullscreenUrl function
func TestPrevFullscreenUrl_MiddleElement_ReturnsPrevious(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
		createTestFile(2, "c.jpg", Image, time.Now(), false),
	}

	result := prevFullscreenUrl(files, 1, "path", "")
	assert.Equal(t, "/fullscreen/0/path", result)
}

func TestPrevFullscreenUrl_FirstElement_WrapsToLast(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
		createTestFile(2, "c.jpg", Image, time.Now(), false),
	}

	result := prevFullscreenUrl(files, 0, "path", "")
	assert.Equal(t, "/fullscreen/2/path", result)
}

// Test Context.getDirectoryById
func TestGetDirectoryById_ValidId_ReturnsDirectory(t *testing.T) {
	dir1 := createTestDirectory(0, "dir1", []int{}, time.Now())
	dir2 := createTestDirectory(1, "dir2", []int{}, time.Now())
	cx := createTestContext([]File{}, []Directory{dir1, dir2})

	result, err := cx.getDirectoryById(1)
	assert.NoError(t, err)
	assert.Equal(t, "dir2", result.name)
	assert.Equal(t, 1, result.id)
}

func TestGetDirectoryById_NegativeId_ReturnsError(t *testing.T) {
	cx := createTestContext([]File{}, []Directory{})

	result, err := cx.getDirectoryById(-1)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetDirectoryById_IdExceedsLength_ReturnsError(t *testing.T) {
	dir := createTestDirectory(0, "dir1", []int{}, time.Now())
	cx := createTestContext([]File{}, []Directory{dir})

	result, err := cx.getDirectoryById(99)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// Test Context.getFileById
func TestGetFileById_ValidId_ReturnsFile(t *testing.T) {
	file1 := createTestFile(0, "a.jpg", Image, time.Now(), false)
	file2 := createTestFile(1, "b.jpg", Image, time.Now(), false)
	cx := createTestContext([]File{file1, file2}, []Directory{})

	result, err := cx.getFileById(1)
	assert.NoError(t, err)
	assert.Equal(t, "b.jpg", result.name)
	assert.Equal(t, 1, result.id)
}

func TestGetFileById_NegativeId_ReturnsError(t *testing.T) {
	cx := createTestContext([]File{}, []Directory{})

	result, err := cx.getFileById(-1)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetFileById_IdExceedsLength_ReturnsError(t *testing.T) {
	file := createTestFile(0, "a.jpg", Image, time.Now(), false)
	cx := createTestContext([]File{file}, []Directory{})

	result, err := cx.getFileById(99)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// Test countMediaFiles
func TestCountMediaFiles_OnlyMediaFiles_ReturnsCorrectCount(t *testing.T) {
	file1 := createTestFile(0, "a.jpg", Image, time.Now(), false)
	file2 := createTestFile(1, "b.mp4", Video, time.Now(), false)
	dir := createTestDirectory(0, "dir", []int{0, 1}, time.Now())
	cx := createTestContext([]File{file1, file2}, []Directory{dir})

	count := countMediaFiles(cx, &dir)
	assert.Equal(t, 2, count)
}

func TestCountMediaFiles_MixedMediaAndOther_CountsOnlyMedia(t *testing.T) {
	file1 := createTestFile(0, "a.jpg", Image, time.Now(), false)
	file2 := createTestFile(1, "doc.pdf", Other, time.Now(), false)
	file3 := createTestFile(2, "b.mp4", Video, time.Now(), false)
	dir := createTestDirectory(0, "dir", []int{0, 1, 2}, time.Now())
	cx := createTestContext([]File{file1, file2, file3}, []Directory{dir})

	count := countMediaFiles(cx, &dir)
	assert.Equal(t, 2, count)
}

func TestCountMediaFiles_WithDeletedFiles_ExcludesDeleted(t *testing.T) {
	file1 := createTestFile(0, "a.jpg", Image, time.Now(), false)
	file2 := createTestFile(1, "b.jpg", Image, time.Now(), true) // deleted
	dir := createTestDirectory(0, "dir", []int{0, 1}, time.Now())
	cx := createTestContext([]File{file1, file2}, []Directory{dir})

	count := countMediaFiles(cx, &dir)
	assert.Equal(t, 1, count)
}

func TestCountMediaFiles_EmptyDirectory_ReturnsZero(t *testing.T) {
	dir := createTestDirectory(0, "dir", []int{}, time.Now())
	cx := createTestContext([]File{}, []Directory{dir})

	count := countMediaFiles(cx, &dir)
	assert.Equal(t, 0, count)
}

// Test getSortedMediaFiles
func TestGetSortedMediaFiles_SortByName_ReturnsSortedAlphabetically(t *testing.T) {
	file1 := createTestFile(0, "c.jpg", Image, time.Now(), false)
	file2 := createTestFile(1, "a.jpg", Image, time.Now(), false)
	file3 := createTestFile(2, "B.jpg", Image, time.Now(), false) // uppercase
	cx := createTestContext([]File{file1, file2, file3}, []Directory{})

	result := getSortedMediaFiles(cx, []int{0, 1, 2}, "name")
	assert.Len(t, result, 3)
	assert.Equal(t, "a.jpg", result[0].name)
	assert.Equal(t, "B.jpg", result[1].name) // case-insensitive
	assert.Equal(t, "c.jpg", result[2].name)
}

func TestGetSortedMediaFiles_SortByLatest_ReturnsSortedByModTimeDesc(t *testing.T) {
	old := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	mid := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	new := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

	file1 := createTestFile(0, "a.jpg", Image, mid, false)
	file2 := createTestFile(1, "b.jpg", Image, old, false)
	file3 := createTestFile(2, "c.jpg", Image, new, false)
	cx := createTestContext([]File{file1, file2, file3}, []Directory{})

	result := getSortedMediaFiles(cx, []int{0, 1, 2}, "latest")
	assert.Len(t, result, 3)
	assert.Equal(t, "c.jpg", result[0].name) // newest
	assert.Equal(t, "a.jpg", result[1].name)
	assert.Equal(t, "b.jpg", result[2].name) // oldest
}

func TestGetSortedMediaFiles_SortByOldest_ReturnsSortedByModTimeAsc(t *testing.T) {
	old := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	mid := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	new := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

	file1 := createTestFile(0, "a.jpg", Image, mid, false)
	file2 := createTestFile(1, "b.jpg", Image, old, false)
	file3 := createTestFile(2, "c.jpg", Image, new, false)
	cx := createTestContext([]File{file1, file2, file3}, []Directory{})

	result := getSortedMediaFiles(cx, []int{0, 1, 2}, "oldest")
	assert.Len(t, result, 3)
	assert.Equal(t, "b.jpg", result[0].name) // oldest
	assert.Equal(t, "a.jpg", result[1].name)
	assert.Equal(t, "c.jpg", result[2].name) // newest
}

func TestGetSortedMediaFiles_FiltersOtherKind_ExcludesNonMedia(t *testing.T) {
	file1 := createTestFile(0, "a.jpg", Image, time.Now(), false)
	file2 := createTestFile(1, "doc.pdf", Other, time.Now(), false)
	file3 := createTestFile(2, "b.mp4", Video, time.Now(), false)
	cx := createTestContext([]File{file1, file2, file3}, []Directory{})

	result := getSortedMediaFiles(cx, []int{0, 1, 2}, "name")
	assert.Len(t, result, 2)
	assert.Equal(t, "a.jpg", result[0].name)
	assert.Equal(t, "b.mp4", result[1].name)
}

func TestGetSortedMediaFiles_FiltersDeleted_ExcludesDeletedFiles(t *testing.T) {
	file1 := createTestFile(0, "a.jpg", Image, time.Now(), false)
	file2 := createTestFile(1, "b.jpg", Image, time.Now(), true) // deleted
	file3 := createTestFile(2, "c.jpg", Image, time.Now(), false)
	cx := createTestContext([]File{file1, file2, file3}, []Directory{})

	result := getSortedMediaFiles(cx, []int{0, 1, 2}, "name")
	assert.Len(t, result, 2)
	assert.Equal(t, "a.jpg", result[0].name)
	assert.Equal(t, "c.jpg", result[1].name)
}

func TestGetSortedMediaFiles_EmptyInput_ReturnsEmptySlice(t *testing.T) {
	cx := createTestContext([]File{}, []Directory{})
	result := getSortedMediaFiles(cx, []int{}, "name")
	assert.Empty(t, result)
}

func TestGetSortedMediaFiles_InvalidSortParam_DefaultsToName(t *testing.T) {
	file1 := createTestFile(0, "c.jpg", Image, time.Now(), false)
	file2 := createTestFile(1, "a.jpg", Image, time.Now(), false)
	cx := createTestContext([]File{file1, file2}, []Directory{})

	result := getSortedMediaFiles(cx, []int{0, 1}, "invalid")
	assert.Len(t, result, 2)
	assert.Equal(t, "a.jpg", result[0].name)
	assert.Equal(t, "c.jpg", result[1].name)
}

// Test findNextNonDeletedById
func TestFindNextNonDeletedById_HasNextWithHigherId_ReturnsNext(t *testing.T) {
	file1 := createTestFile(0, "a.jpg", Image, time.Now(), false)
	file2 := createTestFile(1, "b.jpg", Image, time.Now(), false)
	file3 := createTestFile(2, "c.jpg", Image, time.Now(), false)
	cx := createTestContext([]File{file1, file2, file3}, []Directory{})

	result, found := findNextNonDeletedById(cx, []int{0, 1, 2}, 0)
	assert.True(t, found)
	assert.Equal(t, 1, result.id)
}

func TestFindNextNonDeletedById_NoNextHigherId_WrapsToSmallest(t *testing.T) {
	file1 := createTestFile(0, "a.jpg", Image, time.Now(), false)
	file2 := createTestFile(1, "b.jpg", Image, time.Now(), false)
	file3 := createTestFile(2, "c.jpg", Image, time.Now(), false)
	cx := createTestContext([]File{file1, file2, file3}, []Directory{})

	result, found := findNextNonDeletedById(cx, []int{0, 1, 2}, 2)
	assert.True(t, found)
	assert.Equal(t, 0, result.id) // wraps to smallest
}

func TestFindNextNonDeletedById_AllDeleted_ReturnsNotFound(t *testing.T) {
	file1 := createTestFile(0, "a.jpg", Image, time.Now(), true)
	file2 := createTestFile(1, "b.jpg", Image, time.Now(), true)
	cx := createTestContext([]File{file1, file2}, []Directory{})

	_, found := findNextNonDeletedById(cx, []int{0, 1}, 0)
	assert.False(t, found)
}

func TestFindNextNonDeletedById_AllOtherKind_ReturnsNotFound(t *testing.T) {
	file1 := createTestFile(0, "a.pdf", Other, time.Now(), false)
	file2 := createTestFile(1, "b.txt", Other, time.Now(), false)
	cx := createTestContext([]File{file1, file2}, []Directory{})

	_, found := findNextNonDeletedById(cx, []int{0, 1}, 0)
	assert.False(t, found)
}

func TestFindNextNonDeletedById_MixedDeletedAndNonDeleted_SkipsDeleted(t *testing.T) {
	file1 := createTestFile(0, "a.jpg", Image, time.Now(), false)
	file2 := createTestFile(1, "b.jpg", Image, time.Now(), true) // deleted
	file3 := createTestFile(2, "c.jpg", Image, time.Now(), false)
	cx := createTestContext([]File{file1, file2, file3}, []Directory{})

	result, found := findNextNonDeletedById(cx, []int{0, 1, 2}, 0)
	assert.True(t, found)
	assert.Equal(t, 2, result.id) // skips deleted file 1
}

func TestFindNextNonDeletedById_EmptyFilesList_ReturnsNotFound(t *testing.T) {
	cx := createTestContext([]File{}, []Directory{})
	_, found := findNextNonDeletedById(cx, []int{}, 0)
	assert.False(t, found)
}

// Test childDirectoryData
func TestChildDirectoryData_SortByName_ReturnsSortedAlphabetically(t *testing.T) {
	dir1 := createTestDirectory(0, "charlie", []int{}, time.Now())
	dir2 := createTestDirectory(1, "Alpha", []int{}, time.Now())
	dir3 := createTestDirectory(2, "bravo", []int{}, time.Now())
	parent := Directory{
		id:             3,
		name:           "parent",
		files:          []int{},
		childDirectory: []Directory{dir1, dir2, dir3},
		modTime:        time.Now(),
	}

	result := childDirectoryData(&parent, "path", "name")
	assert.Len(t, result, 3)
	assert.Equal(t, "Alpha", result[0].Name)
	assert.Equal(t, "bravo", result[1].Name)
	assert.Equal(t, "charlie", result[2].Name)
}

func TestChildDirectoryData_SortByLatest_ReturnsSortedByModTimeDesc(t *testing.T) {
	old := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	mid := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	new := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

	dir1 := createTestDirectory(0, "a", []int{}, mid)
	dir2 := createTestDirectory(1, "b", []int{}, old)
	dir3 := createTestDirectory(2, "c", []int{}, new)
	parent := Directory{
		id:             3,
		name:           "parent",
		files:          []int{},
		childDirectory: []Directory{dir1, dir2, dir3},
		modTime:        time.Now(),
	}

	result := childDirectoryData(&parent, "path", "latest")
	assert.Len(t, result, 3)
	assert.Equal(t, "c", result[0].Name) // newest
	assert.Equal(t, "a", result[1].Name)
	assert.Equal(t, "b", result[2].Name) // oldest
}

func TestChildDirectoryData_SortByOldest_ReturnsSortedByModTimeAsc(t *testing.T) {
	old := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	mid := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	new := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

	dir1 := createTestDirectory(0, "a", []int{}, mid)
	dir2 := createTestDirectory(1, "b", []int{}, old)
	dir3 := createTestDirectory(2, "c", []int{}, new)
	parent := Directory{
		id:             3,
		name:           "parent",
		files:          []int{},
		childDirectory: []Directory{dir1, dir2, dir3},
		modTime:        time.Now(),
	}

	result := childDirectoryData(&parent, "path", "oldest")
	assert.Len(t, result, 3)
	assert.Equal(t, "b", result[0].Name) // oldest
	assert.Equal(t, "a", result[1].Name)
	assert.Equal(t, "c", result[2].Name) // newest
}

func TestChildDirectoryData_EmptyDirectory_ReturnsEmptySlice(t *testing.T) {
	parent := Directory{
		id:             0,
		name:           "parent",
		files:          []int{},
		childDirectory: []Directory{},
		modTime:        time.Now(),
	}

	result := childDirectoryData(&parent, "path", "name")
	assert.Empty(t, result)
}

func TestChildDirectoryData_VerifiesUrlFormat(t *testing.T) {
	dir := createTestDirectory(0, "child", []int{}, time.Now())
	parent := Directory{
		id:             1,
		name:           "parent",
		files:          []int{},
		childDirectory: []Directory{dir},
		modTime:        time.Now(),
	}

	result := childDirectoryData(&parent, "base", "latest")
	assert.Len(t, result, 1)
	assert.Equal(t, "/files/base/child?sort=latest", result[0].Url)
}

func TestChildDirectoryData_VariousSortOrders_ReturnsSortedDirectories(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	dirs := []Directory{
		createTestDirectory(1, "charlie", []int{}, baseTime.Add(1*time.Hour)),
		createTestDirectory(2, "alpha", []int{}, baseTime.Add(3*time.Hour)),
		createTestDirectory(3, "bravo", []int{}, baseTime.Add(2*time.Hour)),
	}

	parent := createTestDirectory(0, "parent", []int{}, baseTime)
	parent.childDirectory = dirs

	tests := []struct {
		name      string
		sortParam string
		expected  []string
	}{
		{"Name sort", "name", []string{"alpha", "bravo", "charlie"}},
		{"Name sort uppercase", "NAME", []string{"alpha", "bravo", "charlie"}},
		{"Latest sort", "latest", []string{"alpha", "bravo", "charlie"}},
		{"Oldest sort", "oldest", []string{"charlie", "bravo", "alpha"}},
		{"Default (empty)", "", []string{"alpha", "bravo", "charlie"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := childDirectoryData(&parent, "/test", tt.sortParam)

			assert.Len(t, result, 3)
			for i, expected := range tt.expected {
				assert.Equal(t, expected, result[i].Name)
				assert.True(t, validateURL(result[i].Url))
			}
		})
	}
}

// Test buildFilePath function
func TestBuildFilePath_EmptyDirPath_ReturnsFileName(t *testing.T) {
	cx := createTestContext([]File{}, []Directory{})
	file := createTestFile(0, "test.jpg", Image, time.Now(), false)

	result := cx.buildFilePath(file, "")

	assert.Equal(t, "test.jpg", result)
}

func TestBuildFilePath_WithDirPath_ReturnsLinkedPath(t *testing.T) {
	dir1 := createTestDirectory(1, "parent", []int{}, time.Now())
	dir2 := createTestDirectory(2, "child", []int{}, time.Now())
	cx := createTestContext([]File{}, []Directory{{}, dir1, dir2})

	file := createTestFile(0, "test.jpg", Image, time.Now(), false)
	file.dirPath = []int{0, 1, 2}

	result := cx.buildFilePath(file, "name")

	assert.Contains(t, result, `<a href="/files/parent?sort=name">parent</a>`)
	assert.Contains(t, result, `<a href="/files/parent/child?sort=name">child</a>`)
	assert.Contains(t, result, "test.jpg")
	assert.Contains(t, result, " / ")
}

func TestBuildFilePath_NoSortParam_BuildsLinksWithoutSort(t *testing.T) {
	dir1 := createTestDirectory(1, "parent", []int{}, time.Now())
	cx := createTestContext([]File{}, []Directory{{}, dir1})

	file := createTestFile(0, "test.jpg", Image, time.Now(), false)
	file.dirPath = []int{0, 1}

	result := cx.buildFilePath(file, "")

	assert.Contains(t, result, `<a href="/files/parent">parent</a>`)
	assert.NotContains(t, result, "sort=")
}

func TestBuildFilePath_SkipsVirtualRoot_DoesNotIncludeRootInPath(t *testing.T) {
	dir1 := createTestDirectory(1, "media", []int{}, time.Now())
	cx := createTestContext([]File{}, []Directory{{id: 0, name: "$Virtual"}, dir1})

	file := createTestFile(0, "test.jpg", Image, time.Now(), false)
	file.dirPath = []int{0, 1} // Starts with virtual root

	result := cx.buildFilePath(file, "")

	assert.NotContains(t, result, "$Virtual")
	assert.Contains(t, result, `<a href="/files/media">media</a>`)
}

// Test fileDataInRange function
func TestFileDataInRange_EmptyDirectory_ReturnsEmptySlice(t *testing.T) {
	cx := createTestContext([]File{}, []Directory{})
	dir := createTestDirectory(0, "empty", []int{}, time.Now())

	result := fileDataInRange(cx, &dir, "/test", 0, 10, "name")

	assert.Empty(t, result)
}

func TestFileDataInRange_NormalRange_ReturnsCorrectFiles(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
		createTestFile(2, "c.jpg", Image, time.Now(), false),
	}
	dir := createTestDirectory(0, "test", []int{0, 1, 2}, time.Now())
	cx := createTestContext(files, []Directory{dir})

	result := fileDataInRange(cx, &dir, "/test", 0, 2, "name")

	assert.Len(t, result, 2)
	assert.Equal(t, "a.jpg", result[0].Name)
	assert.Equal(t, "b.jpg", result[1].Name)
	assert.False(t, result[0].IsVideo)
}

func TestFileDataInRange_WrapAround_HandlesNegativeIndices(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
		createTestFile(2, "c.jpg", Image, time.Now(), false),
	}
	dir := createTestDirectory(0, "test", []int{0, 1, 2}, time.Now())
	cx := createTestContext(files, []Directory{dir})

	result := fileDataInRange(cx, &dir, "/test", -2, 1, "name")

	assert.Len(t, result, 3)
	// -2 % 3 = 1, -1 % 3 = 2, 0 % 3 = 0
	assert.Equal(t, "b.jpg", result[0].Name)
	assert.Equal(t, "c.jpg", result[1].Name)
	assert.Equal(t, "a.jpg", result[2].Name)
}

func TestFileDataInRange_WrapAround_HandlesLargeIndices(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), false),
		createTestFile(2, "c.jpg", Image, time.Now(), false),
	}
	dir := createTestDirectory(0, "test", []int{0, 1, 2}, time.Now())
	cx := createTestContext(files, []Directory{dir})

	result := fileDataInRange(cx, &dir, "/test", 4, 7, "name")

	assert.Len(t, result, 3)
	// 4 % 3 = 1, 5 % 3 = 2, 6 % 3 = 0
	assert.Equal(t, "b.jpg", result[0].Name)
	assert.Equal(t, "c.jpg", result[1].Name)
	assert.Equal(t, "a.jpg", result[2].Name)
}

func TestFileDataInRange_SkipsDeletedFiles_OnlyIncludesNonDeleted(t *testing.T) {
	files := []File{
		createTestFile(0, "a.jpg", Image, time.Now(), false),
		createTestFile(1, "b.jpg", Image, time.Now(), true), // deleted
		createTestFile(2, "c.jpg", Image, time.Now(), false),
	}
	dir := createTestDirectory(0, "test", []int{0, 1, 2}, time.Now())
	cx := createTestContext(files, []Directory{dir})

	result := fileDataInRange(cx, &dir, "/test", 0, 2, "name")

	// Should only get 2 files (a.jpg and c.jpg), skipping deleted b.jpg
	// Note: getSortedMediaFiles already filters deleted files, so there are only 2 files total
	assert.Len(t, result, 2)
	assert.Equal(t, "a.jpg", result[0].Name)
	assert.Equal(t, "c.jpg", result[1].Name)
}

func TestFileDataInRange_VideoFile_SetsIsVideoTrue(t *testing.T) {
	files := []File{
		createTestFile(0, "video.mp4", Video, time.Now(), false),
	}
	dir := createTestDirectory(0, "test", []int{0}, time.Now())
	cx := createTestContext(files, []Directory{dir})

	result := fileDataInRange(cx, &dir, "/test", 0, 1, "name")

	assert.Len(t, result, 1)
	assert.True(t, result[0].IsVideo)
	assert.Contains(t, result[0].ResourceUrl, "/video/")
}

func TestFileDataInRange_RespectsSortOrder_SortsCorrectly(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	files := []File{
		createTestFile(0, "charlie.jpg", Image, baseTime.Add(1*time.Hour), false),
		createTestFile(1, "alpha.jpg", Image, baseTime.Add(3*time.Hour), false),
		createTestFile(2, "bravo.jpg", Image, baseTime.Add(2*time.Hour), false),
	}
	dir := createTestDirectory(0, "test", []int{0, 1, 2}, time.Now())
	cx := createTestContext(files, []Directory{dir})

	tests := []struct {
		name     string
		sortBy   string
		expected []string
	}{
		{"Name sort", "name", []string{"alpha.jpg", "bravo.jpg", "charlie.jpg"}},
		{"Latest sort", "latest", []string{"alpha.jpg", "bravo.jpg", "charlie.jpg"}},
		{"Oldest sort", "oldest", []string{"charlie.jpg", "bravo.jpg", "alpha.jpg"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fileDataInRange(cx, &dir, "/test", 0, 3, tt.sortBy)

			assert.Len(t, result, 3)
			for i, expected := range tt.expected {
				assert.Equal(t, expected, result[i].Name)
			}
		})
	}
}

// Test combineDirectories function
func TestCombineDirectories_EmptySlice_CreatesVirtualDirWithNoChildren(t *testing.T) {
	cx := createTestContext([]File{}, []Directory{})
	dirs := []Directory{}

	result := combineDirectories(cx, dirs)

	assert.Equal(t, "Collections", result.name)
	assert.Empty(t, result.files)
	assert.Empty(t, result.childDirectory)
	assert.Equal(t, 0, result.id)
}

func TestCombineDirectories_MultipleDirectories_CreatesVirtualDirWithChildren(t *testing.T) {
	dir1 := createTestDirectory(0, "media1", []int{}, time.Now())
	dir2 := createTestDirectory(1, "media2", []int{}, time.Now())
	cx := createTestContext([]File{}, []Directory{dir1, dir2})

	result := combineDirectories(cx, []Directory{dir1, dir2})

	assert.Equal(t, "Collections", result.name)
	assert.Empty(t, result.files)
	assert.Len(t, result.childDirectory, 2)
	assert.Equal(t, "media1", result.childDirectory[0].name)
	assert.Equal(t, "media2", result.childDirectory[1].name)
}

func TestCombineDirectories_AddsToContext_IncreasesDirectoryCount(t *testing.T) {
	dir1 := createTestDirectory(0, "media1", []int{}, time.Now())
	cx := createTestContext([]File{}, []Directory{dir1})
	initialLen := len(cx.directories)

	combineDirectories(cx, []Directory{dir1})

	assert.Len(t, cx.directories, initialLen+1)
}

func TestCombineDirectories_AssignsCorrectID_UsesNextAvailableID(t *testing.T) {
	dir1 := createTestDirectory(0, "media1", []int{}, time.Now())
	dir2 := createTestDirectory(1, "media2", []int{}, time.Now())
	cx := createTestContext([]File{}, []Directory{dir1, dir2})

	result := combineDirectories(cx, []Directory{dir1, dir2})

	assert.Equal(t, 2, result.id) // Next available ID after 0 and 1
}

// Test fileResourceURL panic case
func TestFileResourceURL_OtherFileKind_Panics(t *testing.T) {
	file := createTestFile(0, "document.pdf", Other, time.Now(), false)

	assert.Panics(t, func() {
		fileResourceURL(file)
	})
}

// Test getDirectoryByPath function
func TestGetDirectoryByPath_EmptyPath_ReturnsRoot(t *testing.T) {
	root := createTestDirectory(0, "root", []int{}, time.Now())

	result := getDirectoryByPath(&root, "")

	assert.NotNil(t, result)
	assert.Equal(t, "root", result.name)
	assert.Equal(t, 0, result.id)
}

func TestGetDirectoryByPath_RootWithSlash_ReturnsRoot(t *testing.T) {
	root := createTestDirectory(0, "root", []int{}, time.Now())

	result := getDirectoryByPath(&root, "/")

	assert.NotNil(t, result)
	assert.Equal(t, "root", result.name)
}

func TestGetDirectoryByPath_SingleLevel_ReturnsChild(t *testing.T) {
	child := createTestDirectory(1, "child", []int{}, time.Now())
	root := createTestDirectory(0, "root", []int{}, time.Now())
	root.childDirectory = []Directory{child}

	result := getDirectoryByPath(&root, "child")

	assert.NotNil(t, result)
	assert.Equal(t, "child", result.name)
	assert.Equal(t, 1, result.id)
}

func TestGetDirectoryByPath_SingleLevelWithSlash_ReturnsChild(t *testing.T) {
	child := createTestDirectory(1, "child", []int{}, time.Now())
	root := createTestDirectory(0, "root", []int{}, time.Now())
	root.childDirectory = []Directory{child}

	result := getDirectoryByPath(&root, "/child/")

	assert.NotNil(t, result)
	assert.Equal(t, "child", result.name)
}

func TestGetDirectoryByPath_MultiLevel_ReturnsDeepChild(t *testing.T) {
	grandchild := createTestDirectory(2, "grandchild", []int{}, time.Now())
	child := createTestDirectory(1, "child", []int{}, time.Now())
	child.childDirectory = []Directory{grandchild}
	root := createTestDirectory(0, "root", []int{}, time.Now())
	root.childDirectory = []Directory{child}

	result := getDirectoryByPath(&root, "child/grandchild")

	assert.NotNil(t, result)
	assert.Equal(t, "grandchild", result.name)
	assert.Equal(t, 2, result.id)
}

func TestGetDirectoryByPath_MultiLevelWithSlashes_ReturnsDeepChild(t *testing.T) {
	grandchild := createTestDirectory(2, "grandchild", []int{}, time.Now())
	child := createTestDirectory(1, "child", []int{}, time.Now())
	child.childDirectory = []Directory{grandchild}
	root := createTestDirectory(0, "root", []int{}, time.Now())
	root.childDirectory = []Directory{child}

	result := getDirectoryByPath(&root, "/child/grandchild/")

	assert.NotNil(t, result)
	assert.Equal(t, "grandchild", result.name)
}

func TestGetDirectoryByPath_NonExistentPath_ReturnsNil(t *testing.T) {
	root := createTestDirectory(0, "root", []int{}, time.Now())

	result := getDirectoryByPath(&root, "nonexistent")

	assert.Nil(t, result)
}

func TestGetDirectoryByPath_PartiallyCorrectPath_ReturnsNil(t *testing.T) {
	child := createTestDirectory(1, "child", []int{}, time.Now())
	root := createTestDirectory(0, "root", []int{}, time.Now())
	root.childDirectory = []Directory{child}

	result := getDirectoryByPath(&root, "child/nonexistent")

	assert.Nil(t, result)
}

func TestGetDirectoryByPath_MultipleSiblings_ReturnsCorrectOne(t *testing.T) {
	child1 := createTestDirectory(1, "alpha", []int{}, time.Now())
	child2 := createTestDirectory(2, "beta", []int{}, time.Now())
	child3 := createTestDirectory(3, "gamma", []int{}, time.Now())
	root := createTestDirectory(0, "root", []int{}, time.Now())
	root.childDirectory = []Directory{child1, child2, child3}

	result := getDirectoryByPath(&root, "beta")

	assert.NotNil(t, result)
	assert.Equal(t, "beta", result.name)
	assert.Equal(t, 2, result.id)
}

func TestGetDirectoryByPath_DeepNesting_NavigatesCorrectly(t *testing.T) {
	level3 := createTestDirectory(3, "level3", []int{}, time.Now())
	level2 := createTestDirectory(2, "level2", []int{}, time.Now())
	level2.childDirectory = []Directory{level3}
	level1 := createTestDirectory(1, "level1", []int{}, time.Now())
	level1.childDirectory = []Directory{level2}
	root := createTestDirectory(0, "root", []int{}, time.Now())
	root.childDirectory = []Directory{level1}

	result := getDirectoryByPath(&root, "level1/level2/level3")

	assert.NotNil(t, result)
	assert.Equal(t, "level3", result.name)
	assert.Equal(t, 3, result.id)
}

func TestGetDirectoryByPath_CaseSensitive_ReturnsNilForWrongCase(t *testing.T) {
	child := createTestDirectory(1, "Child", []int{}, time.Now())
	root := createTestDirectory(0, "root", []int{}, time.Now())
	root.childDirectory = []Directory{child}

	result := getDirectoryByPath(&root, "child") // lowercase

	assert.Nil(t, result) // Should be nil because Go is case-sensitive
}
