package main

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Test parseDeleteReq function
func TestParseDeleteReq_ValidInput_ParsesCorrectly(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	formData := url.Values{}
	formData.Set("fileId", "42")
	formData.Set("directoryId", "10")
	formData.Set("sort", "latest")

	c.Request = httptest.NewRequest("POST", "/delete", strings.NewReader(formData.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := parseDeleteReq(c)

	assert.NoError(t, err)
	assert.Equal(t, 42, result.fileId)
	assert.Equal(t, 10, result.dirId)
	assert.Equal(t, Latest, result.sortBy)
}

func TestParseDeleteReq_DefaultSort_UsesName(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	formData := url.Values{}
	formData.Set("fileId", "5")
	formData.Set("directoryId", "3")

	c.Request = httptest.NewRequest("POST", "/delete", strings.NewReader(formData.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := parseDeleteReq(c)

	assert.NoError(t, err)
	assert.Equal(t, 5, result.fileId)
	assert.Equal(t, 3, result.dirId)
	assert.Equal(t, Name, result.sortBy)
}

func TestParseDeleteReq_OldestSort_ParsesCorrectly(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	formData := url.Values{}
	formData.Set("fileId", "15")
	formData.Set("directoryId", "7")
	formData.Set("sort", "oldest")

	c.Request = httptest.NewRequest("POST", "/delete", strings.NewReader(formData.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := parseDeleteReq(c)

	assert.NoError(t, err)
	assert.Equal(t, 15, result.fileId)
	assert.Equal(t, 7, result.dirId)
	assert.Equal(t, Oldest, result.sortBy)
}

func TestParseDeleteReq_InvalidFileId_ReturnsError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	formData := url.Values{}
	formData.Set("fileId", "invalid")
	formData.Set("directoryId", "10")

	c.Request = httptest.NewRequest("POST", "/delete", strings.NewReader(formData.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := parseDeleteReq(c)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestParseDeleteReq_InvalidDirectoryId_ReturnsError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	formData := url.Values{}
	formData.Set("fileId", "42")
	formData.Set("directoryId", "invalid")

	c.Request = httptest.NewRequest("POST", "/delete", strings.NewReader(formData.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := parseDeleteReq(c)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestParseDeleteReq_InvalidSortParam_ReturnsError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	formData := url.Values{}
	formData.Set("fileId", "42")
	formData.Set("directoryId", "10")
	formData.Set("sort", "invalid")

	c.Request = httptest.NewRequest("POST", "/delete", strings.NewReader(formData.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := parseDeleteReq(c)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestParseDeleteReq_MissingFileId_ReturnsError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	formData := url.Values{}
	formData.Set("directoryId", "10")

	c.Request = httptest.NewRequest("POST", "/delete", strings.NewReader(formData.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := parseDeleteReq(c)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestParseDeleteReq_MissingDirectoryId_ReturnsError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	formData := url.Values{}
	formData.Set("fileId", "42")

	c.Request = httptest.NewRequest("POST", "/delete", strings.NewReader(formData.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := parseDeleteReq(c)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestParseDeleteReq_CaseInsensitiveSort_ParsesCorrectly(t *testing.T) {
	tests := []struct {
		name     string
		sortStr  string
		expected SortBy
	}{
		{"Uppercase LATEST", "LATEST", Latest},
		{"Mixed case Latest", "Latest", Latest},
		{"Uppercase OLDEST", "OLDEST", Oldest},
		{"Mixed case Oldest", "Oldest", Oldest},
		{"Uppercase NAME", "NAME", Name},
		{"lowercase name", "name", Name},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			formData := url.Values{}
			formData.Set("fileId", "1")
			formData.Set("directoryId", "1")
			formData.Set("sort", tt.sortStr)

			c.Request = httptest.NewRequest("POST", "/delete", strings.NewReader(formData.Encode()))
			c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			result, err := parseDeleteReq(c)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result.sortBy)
		})
	}
}

// MockSystemUtils for testing
type MockSystemUtils struct {
	MoveToTrashFunc func(path string) error
	CalledPaths     []string
}

func (m *MockSystemUtils) MoveToTrash(path string) error {
	m.CalledPaths = append(m.CalledPaths, path)
	if m.MoveToTrashFunc != nil {
		return m.MoveToTrashFunc(path)
	}
	return nil
}

// Test NewSystemUtils function
func TestNewSystemUtils_ReturnsCorrectImplementation(t *testing.T) {
	systemUtils := NewSystemUtils()

	// Verify it returns a non-nil SystemUtils
	assert.NotNil(t, systemUtils)

	// Verify it implements the SystemUtils interface
	var _ SystemUtils = systemUtils

	// Test that it can be called (though we can't easily test the specific OS without mocking runtime.GOOS)
	err := systemUtils.MoveToTrash("/tmp/test")
	// We expect this to fail since /tmp/test doesn't exist, but it should not panic
	assert.Error(t, err)
}

// Test handleDelete function
func TestHandleDelete_SuccessfulDeletion_ReturnsNextFile(t *testing.T) {
	// Setup test data
	mockSystemUtils := &MockSystemUtils{}
	cx := &Context{
		directories: []Directory{
			{id: 0, name: "root", files: []int{0, 1, 2}},
		},
		files: []File{
			{id: 0, name: "file1.jpg", kind: Image, filePath: "/path/file1.jpg", deleted: false},
			{id: 1, name: "file2.jpg", kind: Image, filePath: "/path/file2.jpg", deleted: false},
			{id: 2, name: "file3.jpg", kind: Image, filePath: "/path/file3.jpg", deleted: false},
		},
		systemUtils: mockSystemUtils,
	}

	req := deleteReq{
		ReqBase: ReqBase{sortBy: Name},
		fileId:  0,
		dirId:   0,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Execute
	result, err := handleDelete(cx, c, req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.id)       // Should return next file (file2.jpg)
	assert.True(t, cx.files[0].deleted) // File should be marked as deleted
	assert.Len(t, mockSystemUtils.CalledPaths, 1)
	assert.Equal(t, "/path/file1.jpg", mockSystemUtils.CalledPaths[0])
}

func TestHandleDelete_LastFileInDirectory_WrapsToFirst(t *testing.T) {
	// Setup test data - file2 is the last file
	mockSystemUtils := &MockSystemUtils{}
	cx := &Context{
		directories: []Directory{
			{id: 0, name: "root", files: []int{0, 1, 2}},
		},
		files: []File{
			{id: 0, name: "file1.jpg", kind: Image, filePath: "/path/file1.jpg", deleted: false},
			{id: 1, name: "file2.jpg", kind: Image, filePath: "/path/file2.jpg", deleted: false},
			{id: 2, name: "file3.jpg", kind: Image, filePath: "/path/file3.jpg", deleted: false},
		},
		systemUtils: mockSystemUtils,
	}

	req := deleteReq{
		ReqBase: ReqBase{sortBy: Name},
		fileId:  2, // Delete last file
		dirId:   0,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Execute
	result, err := handleDelete(cx, c, req)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.id)       // Should wrap to first file (file1.jpg)
	assert.True(t, cx.files[2].deleted) // File should be marked as deleted
	assert.Len(t, mockSystemUtils.CalledPaths, 1)
	assert.Equal(t, "/path/file3.jpg", mockSystemUtils.CalledPaths[0])
}

func TestHandleDelete_FileNotFound_ReturnsError(t *testing.T) {
	mockSystemUtils := &MockSystemUtils{}
	cx := &Context{
		directories: []Directory{
			{id: 0, name: "root", files: []int{0, 1}},
		},
		files: []File{
			{id: 0, name: "file1.jpg", kind: Image, filePath: "/path/file1.jpg", deleted: false},
			{id: 1, name: "file2.jpg", kind: Image, filePath: "/path/file2.jpg", deleted: false},
		},
		systemUtils: mockSystemUtils,
	}

	req := deleteReq{
		ReqBase: ReqBase{sortBy: Name},
		fileId:  5, // Non-existent file ID
		dirId:   0,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Execute
	result, err := handleDelete(cx, c, req)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid file id 5")
	assert.Len(t, mockSystemUtils.CalledPaths, 0) // Should not be called
}

func TestHandleDelete_DirectoryNotFound_ReturnsError(t *testing.T) {
	mockSystemUtils := &MockSystemUtils{}
	cx := &Context{
		directories: []Directory{
			{id: 0, name: "root", files: []int{0, 1}},
		},
		files: []File{
			{id: 0, name: "file1.jpg", kind: Image, filePath: "/path/file1.jpg", deleted: false},
			{id: 1, name: "file2.jpg", kind: Image, filePath: "/path/file2.jpg", deleted: false},
		},
		systemUtils: mockSystemUtils,
	}

	req := deleteReq{
		ReqBase: ReqBase{sortBy: Name},
		fileId:  0,
		dirId:   5, // Non-existent directory ID
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Execute
	result, err := handleDelete(cx, c, req)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid directory id 5")
	assert.Len(t, mockSystemUtils.CalledPaths, 0) // Should not be called
}

func TestHandleDelete_AlreadyDeletedFile_ReturnsError(t *testing.T) {
	mockSystemUtils := &MockSystemUtils{}
	cx := &Context{
		directories: []Directory{
			{id: 0, name: "root", files: []int{0, 1}},
		},
		files: []File{
			{id: 0, name: "file1.jpg", kind: Image, filePath: "/path/file1.jpg", deleted: true}, // Already deleted
			{id: 1, name: "file2.jpg", kind: Image, filePath: "/path/file2.jpg", deleted: false},
		},
		systemUtils: mockSystemUtils,
	}

	req := deleteReq{
		ReqBase: ReqBase{sortBy: Name},
		fileId:  0,
		dirId:   0,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Execute
	result, err := handleDelete(cx, c, req)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "trying to delete already deleted file at 0")
	assert.Len(t, mockSystemUtils.CalledPaths, 0) // Should not be called
}

func TestHandleDelete_TrashMoverFails_ReturnsError(t *testing.T) {
	mockSystemUtils := &MockSystemUtils{
		MoveToTrashFunc: func(path string) error {
			return fmt.Errorf("trash failed")
		},
	}
	cx := &Context{
		directories: []Directory{
			{id: 0, name: "root", files: []int{0, 1}},
		},
		files: []File{
			{id: 0, name: "file1.jpg", kind: Image, filePath: "/path/file1.jpg", deleted: false},
			{id: 1, name: "file2.jpg", kind: Image, filePath: "/path/file2.jpg", deleted: false},
		},
		systemUtils: mockSystemUtils,
	}

	req := deleteReq{
		ReqBase: ReqBase{sortBy: Name},
		fileId:  0,
		dirId:   0,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Execute
	result, err := handleDelete(cx, c, req)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to delete file 0 due to trash failed")
	assert.Len(t, mockSystemUtils.CalledPaths, 1)
	assert.Equal(t, "/path/file1.jpg", mockSystemUtils.CalledPaths[0])
	assert.True(t, cx.files[0].deleted) // File is marked as deleted before calling trash
}

func TestHandleDelete_FileNotFoundInDirectory_ReturnsError(t *testing.T) {
	// Setup where file exists but is not in the specified directory's file list
	mockSystemUtils := &MockSystemUtils{}
	cx := &Context{
		directories: []Directory{
			{id: 0, name: "root", files: []int{1, 2}}, // file 0 is not in this directory
		},
		files: []File{
			{id: 0, name: "file1.jpg", kind: Image, filePath: "/path/file1.jpg", deleted: false},
			{id: 1, name: "file2.jpg", kind: Image, filePath: "/path/file2.jpg", deleted: false},
			{id: 2, name: "file3.jpg", kind: Image, filePath: "/path/file3.jpg", deleted: false},
		},
		systemUtils: mockSystemUtils,
	}

	req := deleteReq{
		ReqBase: ReqBase{sortBy: Name},
		fileId:  0, // File exists but not in directory
		dirId:   0,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Execute
	result, err := handleDelete(cx, c, req)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to find file in directory 0")
	assert.Len(t, mockSystemUtils.CalledPaths, 0) // Should not be called
}

func TestHandleDelete_WithDifferentSortOrders_ReturnsCorrectNextFile(t *testing.T) {
	// Setup test data with files having different modification times
	baseTime := time.Now()
	mockSystemUtils := &MockSystemUtils{}
	cx := &Context{
		directories: []Directory{
			{id: 0, name: "root", files: []int{0, 1, 2}},
		},
		files: []File{
			{id: 0, name: "file1.jpg", kind: Image, filePath: "/path/file1.jpg", deleted: false, modTime: baseTime.Add(-2 * time.Hour)},
			{id: 1, name: "file2.jpg", kind: Image, filePath: "/path/file2.jpg", deleted: false, modTime: baseTime.Add(-1 * time.Hour)},
			{id: 2, name: "file3.jpg", kind: Image, filePath: "/path/file3.jpg", deleted: false, modTime: baseTime},
		},
		systemUtils: mockSystemUtils,
	}

	tests := []struct {
		name     string
		sortBy   SortBy
		fileId   int
		expected int
	}{
		{"Name sort - delete first", Name, 0, 1},
		{"Name sort - delete middle", Name, 1, 2},
		{"Name sort - delete last", Name, 2, 0},
		{"Latest sort - delete newest", Latest, 2, 1},
		{"Latest sort - delete middle", Latest, 1, 0},
		{"Latest sort - delete oldest", Latest, 0, 2},
		{"Oldest sort - delete oldest", Oldest, 0, 1},
		{"Oldest sort - delete middle", Oldest, 1, 2},
		{"Oldest sort - delete newest", Oldest, 2, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset file deleted status
			for i := range cx.files {
				cx.files[i].deleted = false
			}

			// Reset mock for each test iteration
			mockSystemUtils.CalledPaths = []string{}

			req := deleteReq{
				ReqBase: ReqBase{sortBy: tt.sortBy},
				fileId:  tt.fileId,
				dirId:   0,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Execute
			result, err := handleDelete(cx, c, req)

			// Verify
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, result.id)
			assert.True(t, cx.files[tt.fileId].deleted)
			assert.Len(t, mockSystemUtils.CalledPaths, 1)
		})
	}
}
