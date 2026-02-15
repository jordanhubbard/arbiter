package files

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type staticResolver struct {
	dir string
}

func (r staticResolver) GetProjectWorkDir(projectID string) string {
	return r.dir
}

type emptyResolver struct{}

func (r emptyResolver) GetProjectWorkDir(projectID string) string {
	return ""
}

// --- NewManager ---

func TestNewManager(t *testing.T) {
	mgr := NewManager(staticResolver{dir: "/tmp"})
	if mgr == nil {
		t.Fatal("Expected non-nil manager")
	}
	if mgr.WorkDirs == nil {
		t.Error("Expected non-nil WorkDirs")
	}
}

func TestNewManager_NilResolver(t *testing.T) {
	mgr := NewManager(nil)
	if mgr == nil {
		t.Fatal("Expected non-nil manager")
	}
}

// --- ReadFile ---

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	mgr := NewManager(staticResolver{dir: dir})
	res, err := mgr.ReadFile(context.Background(), "proj-1", "README.md")
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if res.Content != "hello" {
		t.Fatalf("unexpected content: %s", res.Content)
	}
	if res.Path != "README.md" {
		t.Errorf("expected path 'README.md', got '%s'", res.Path)
	}
	if res.Size != 5 {
		t.Errorf("expected size 5, got %d", res.Size)
	}
}

func TestReadFilePathTraversal(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	if _, err := mgr.ReadFile(context.Background(), "proj-1", "../secret.txt"); err == nil {
		t.Fatalf("expected path traversal error")
	}
}

func TestReadFile_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.ReadFile(context.Background(), "proj-1", "/etc/passwd")
	if err == nil {
		t.Fatal("Expected error for absolute path")
	}
	if !strings.Contains(err.Error(), "path must be relative") {
		t.Errorf("Expected 'path must be relative' error, got: %v", err)
	}
}

func TestReadFile_IsDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.ReadFile(context.Background(), "proj-1", "subdir")
	if err == nil {
		t.Fatal("Expected error for directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("Expected directory error, got: %v", err)
	}
}

func TestReadFile_NonExistent(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.ReadFile(context.Background(), "proj-1", "nofile.txt")
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}
}

func TestReadFile_BlockedGitPath(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.ReadFile(context.Background(), "proj-1", ".git/config")
	if err == nil {
		t.Fatal("Expected error for blocked .git path")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("Expected 'not allowed' error, got: %v", err)
	}
}

func TestReadFile_NilResolver(t *testing.T) {
	mgr := NewManager(nil)
	_, err := mgr.ReadFile(context.Background(), "proj-1", "test.txt")
	if err == nil {
		t.Fatal("Expected error with nil resolver")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("Expected 'not configured' error, got: %v", err)
	}
}

func TestReadFile_EmptyWorkDir(t *testing.T) {
	mgr := NewManager(emptyResolver{})
	_, err := mgr.ReadFile(context.Background(), "proj-1", "test.txt")
	if err == nil {
		t.Fatal("Expected error with empty workdir")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// --- ReadTree ---

func TestReadTree(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(staticResolver{dir: dir})
	entries, err := mgr.ReadTree(context.Background(), "proj-1", ".", 10, 100)
	if err != nil {
		t.Fatalf("ReadTree: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("Expected at least 2 entries, got %d", len(entries))
	}

	// Check that we find both a file and a directory
	foundFile := false
	foundDir := false
	for _, e := range entries {
		if e.Type == "file" {
			foundFile = true
		}
		if e.Type == "dir" {
			foundDir = true
		}
	}
	if !foundFile {
		t.Error("Expected at least one file entry")
	}
	if !foundDir {
		t.Error("Expected at least one dir entry")
	}
}

func TestReadTree_EmptyRelPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(staticResolver{dir: dir})
	entries, err := mgr.ReadTree(context.Background(), "proj-1", "", 10, 100)
	if err != nil {
		t.Fatalf("ReadTree: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("Expected entries for empty rel path")
	}
}

func TestReadTree_DefaultDepthAndLimit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("f"), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(staticResolver{dir: dir})
	entries, err := mgr.ReadTree(context.Background(), "proj-1", ".", 0, 0)
	if err != nil {
		t.Fatalf("ReadTree: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("Expected at least 1 entry")
	}
}

func TestReadTree_DepthLimit(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b", "c", "d")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "deep.txt"), []byte("deep"), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(staticResolver{dir: dir})
	entries, err := mgr.ReadTree(context.Background(), "proj-1", ".", 1, 100)
	if err != nil {
		t.Fatalf("ReadTree: %v", err)
	}
	// With maxDepth=1, we should only see the top level "a" directory
	for _, e := range entries {
		if e.Depth > 1 {
			t.Errorf("Found entry at depth %d, expected max 1: %s", e.Depth, e.Path)
		}
	}
}

func TestReadTree_LimitEntries(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 10; i++ {
		name := filepath.Join(dir, strings.Repeat("f", i+1)+".txt")
		if err := os.WriteFile(name, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	mgr := NewManager(staticResolver{dir: dir})
	entries, err := mgr.ReadTree(context.Background(), "proj-1", ".", 10, 3)
	if err != nil {
		t.Fatalf("ReadTree: %v", err)
	}
	if len(entries) > 3 {
		t.Errorf("Expected at most 3 entries, got %d", len(entries))
	}
}

func TestReadTree_BlockedGitDir(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(staticResolver{dir: dir})
	entries, err := mgr.ReadTree(context.Background(), "proj-1", ".", 10, 100)
	if err != nil {
		t.Fatalf("ReadTree: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Path, ".git") {
			t.Errorf("Found .git entry in tree: %s", e.Path)
		}
	}
}

func TestReadTree_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.ReadTree(context.Background(), "proj-1", "../..", 10, 100)
	if err == nil {
		t.Fatal("Expected error for path traversal in ReadTree")
	}
}

// --- SearchText ---

func TestSearchText(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n// TODO\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	mgr := NewManager(staticResolver{dir: dir})
	results, err := mgr.SearchText(context.Background(), "proj-1", ".", "TODO", 10)
	if err != nil {
		t.Fatalf("search text: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 match, got %d", len(results))
	}
	if results[0].Line != 2 {
		t.Errorf("expected line 2, got %d", results[0].Line)
	}
	if results[0].Text != "// TODO" {
		t.Errorf("expected text '// TODO', got '%s'", results[0].Text)
	}
}

func TestSearchText_EmptyQuery(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.SearchText(context.Background(), "proj-1", ".", "", 10)
	if err == nil {
		t.Fatal("Expected error for empty query")
	}
	if !strings.Contains(err.Error(), "query is required") {
		t.Errorf("Expected 'query is required' error, got: %v", err)
	}
}

func TestSearchText_WhitespaceQuery(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.SearchText(context.Background(), "proj-1", ".", "   ", 10)
	if err == nil {
		t.Fatal("Expected error for whitespace-only query")
	}
}

func TestSearchText_EmptyRelPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("findme\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(staticResolver{dir: dir})
	results, err := mgr.SearchText(context.Background(), "proj-1", "", "findme", 10)
	if err != nil {
		t.Fatalf("SearchText: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
}

func TestSearchText_Limit(t *testing.T) {
	dir := t.TempDir()
	content := ""
	for i := 0; i < 20; i++ {
		content += "match_this\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "big.txt"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(staticResolver{dir: dir})
	results, err := mgr.SearchText(context.Background(), "proj-1", ".", "match_this", 5)
	if err != nil {
		t.Fatalf("SearchText: %v", err)
	}
	if len(results) > 5 {
		t.Errorf("Expected at most 5 results, got %d", len(results))
	}
}

func TestSearchText_DefaultLimit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(staticResolver{dir: dir})
	results, err := mgr.SearchText(context.Background(), "proj-1", ".", "hello", 0)
	if err != nil {
		t.Fatalf("SearchText: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestSearchText_NoMatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("nothing here\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(staticResolver{dir: dir})
	results, err := mgr.SearchText(context.Background(), "proj-1", ".", "NOMATCH", 10)
	if err != nil {
		t.Fatalf("SearchText: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestSearchText_BlockedPath(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.SearchText(context.Background(), "proj-1", "../..", "query", 10)
	if err == nil {
		t.Fatal("Expected error for path traversal")
	}
}

func TestSearchText_SkipsGitDir(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte("findme\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("findme\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(staticResolver{dir: dir})
	results, err := mgr.SearchText(context.Background(), "proj-1", ".", "findme", 100)
	if err != nil {
		t.Fatalf("SearchText: %v", err)
	}
	for _, r := range results {
		if strings.Contains(r.Path, ".git") {
			t.Errorf("Found .git result in search: %s", r.Path)
		}
	}
}

// --- WriteFile ---

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})

	result, err := mgr.WriteFile(context.Background(), "proj-1", "new.txt", "hello world")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if result.Path != "new.txt" {
		t.Errorf("Expected path 'new.txt', got '%s'", result.Path)
	}
	if result.BytesWritten != 11 {
		t.Errorf("Expected 11 bytes, got %d", result.BytesWritten)
	}

	// Verify content
	data, err := os.ReadFile(filepath.Join(dir, "new.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", string(data))
	}
}

func TestWriteFile_EmptyPath(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.WriteFile(context.Background(), "proj-1", "", "content")
	if err == nil {
		t.Fatal("Expected error for empty path")
	}
	if !strings.Contains(err.Error(), "path is required") {
		t.Errorf("Expected 'path is required' error, got: %v", err)
	}
}

func TestWriteFile_WhitespacePath(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.WriteFile(context.Background(), "proj-1", "  ", "content")
	if err == nil {
		t.Fatal("Expected error for whitespace path")
	}
}

func TestWriteFile_BlockedPath(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.WriteFile(context.Background(), "proj-1", ".git/config", "content")
	if err == nil {
		t.Fatal("Expected error for blocked .git path")
	}
}

func TestWriteFile_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.WriteFile(context.Background(), "proj-1", "../escape.txt", "content")
	if err == nil {
		t.Fatal("Expected error for path traversal")
	}
}

func TestWriteFile_CreatesSubdirectory(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})

	_, err := mgr.WriteFile(context.Background(), "proj-1", "sub/dir/file.txt", "data")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "sub", "dir", "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "data" {
		t.Errorf("Expected 'data', got '%s'", string(data))
	}
}

func TestWriteFile_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})

	result, err := mgr.WriteFile(context.Background(), "proj-1", "empty.txt", "")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if result.BytesWritten != 0 {
		t.Errorf("Expected 0 bytes, got %d", result.BytesWritten)
	}
}

func TestWriteFile_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(staticResolver{dir: dir})

	_, err := mgr.WriteFile(context.Background(), "proj-1", "existing.txt", "new content")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "existing.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new content" {
		t.Errorf("Expected 'new content', got '%s'", string(data))
	}
}

// --- MoveFile ---

func TestMoveFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(srcPath, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.MoveFile(context.Background(), "proj-1", "source.txt", "dest.txt")
	if err != nil {
		t.Fatalf("MoveFile: %v", err)
	}

	// Source should not exist
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Error("Source file should not exist after move")
	}
	// Dest should exist
	data, err := os.ReadFile(filepath.Join(dir, "dest.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "data" {
		t.Errorf("Expected 'data', got '%s'", string(data))
	}
}

func TestMoveFile_EmptySourcePath(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.MoveFile(context.Background(), "proj-1", "", "dest.txt")
	if err == nil {
		t.Fatal("Expected error for empty source path")
	}
	if !strings.Contains(err.Error(), "source path is required") {
		t.Errorf("Expected 'source path is required', got: %v", err)
	}
}

func TestMoveFile_EmptyTargetPath(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.MoveFile(context.Background(), "proj-1", "src.txt", "")
	if err == nil {
		t.Fatal("Expected error for empty target path")
	}
	if !strings.Contains(err.Error(), "target path is required") {
		t.Errorf("Expected 'target path is required', got: %v", err)
	}
}

func TestMoveFile_SourceNotFound(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.MoveFile(context.Background(), "proj-1", "nofile.txt", "dest.txt")
	if err == nil {
		t.Fatal("Expected error for non-existent source")
	}
	if !strings.Contains(err.Error(), "source file not found") {
		t.Errorf("Expected 'source file not found', got: %v", err)
	}
}

func TestMoveFile_BlockedSourcePath(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.MoveFile(context.Background(), "proj-1", ".git/config", "dest.txt")
	if err == nil {
		t.Fatal("Expected error for blocked source path")
	}
}

func TestMoveFile_BlockedTargetPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "src.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.MoveFile(context.Background(), "proj-1", "src.txt", ".git/dest.txt")
	if err == nil {
		t.Fatal("Expected error for blocked target path")
	}
}

func TestMoveFile_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.MoveFile(context.Background(), "proj-1", "../escape.txt", "dest.txt")
	if err == nil {
		t.Fatal("Expected error for path traversal in source")
	}
}

func TestMoveFile_CreatesTargetDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "src.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.MoveFile(context.Background(), "proj-1", "src.txt", "newdir/dest.txt")
	if err != nil {
		t.Fatalf("MoveFile: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "newdir", "dest.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "x" {
		t.Errorf("Expected 'x', got '%s'", string(data))
	}
}

// --- DeleteFile ---

func TestDeleteFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "todelete.txt"), []byte("bye"), 0644); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.DeleteFile(context.Background(), "proj-1", "todelete.txt")
	if err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "todelete.txt")); !os.IsNotExist(err) {
		t.Error("File should have been deleted")
	}
}

func TestDeleteFile_EmptyPath(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.DeleteFile(context.Background(), "proj-1", "")
	if err == nil {
		t.Fatal("Expected error for empty path")
	}
	if !strings.Contains(err.Error(), "path is required") {
		t.Errorf("Expected 'path is required', got: %v", err)
	}
}

func TestDeleteFile_NonExistent(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.DeleteFile(context.Background(), "proj-1", "nofile.txt")
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("Expected 'file not found', got: %v", err)
	}
}

func TestDeleteFile_BlockedPath(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.DeleteFile(context.Background(), "proj-1", ".git/config")
	if err == nil {
		t.Fatal("Expected error for blocked path")
	}
}

func TestDeleteFile_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.DeleteFile(context.Background(), "proj-1", "../escape.txt")
	if err == nil {
		t.Fatal("Expected error for path traversal")
	}
}

// --- RenameFile ---

func TestRenameFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "old.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.RenameFile(context.Background(), "proj-1", "old.txt", "new.txt")
	if err != nil {
		t.Fatalf("RenameFile: %v", err)
	}

	// Old should not exist
	if _, err := os.Stat(filepath.Join(dir, "old.txt")); !os.IsNotExist(err) {
		t.Error("Old file should not exist")
	}
	// New should exist
	data, err := os.ReadFile(filepath.Join(dir, "new.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "content" {
		t.Errorf("Expected 'content', got '%s'", string(data))
	}
}

func TestRenameFile_EmptySourcePath(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.RenameFile(context.Background(), "proj-1", "", "new.txt")
	if err == nil {
		t.Fatal("Expected error for empty source path")
	}
	if !strings.Contains(err.Error(), "source path is required") {
		t.Errorf("Expected 'source path is required', got: %v", err)
	}
}

func TestRenameFile_EmptyNewName(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.RenameFile(context.Background(), "proj-1", "old.txt", "")
	if err == nil {
		t.Fatal("Expected error for empty new name")
	}
	if !strings.Contains(err.Error(), "new name is required") {
		t.Errorf("Expected 'new name is required', got: %v", err)
	}
}

func TestRenameFile_NewNameContainsSlash(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.RenameFile(context.Background(), "proj-1", "old.txt", "sub/new.txt")
	if err == nil {
		t.Fatal("Expected error when new name contains slash")
	}
	if !strings.Contains(err.Error(), "filename, not a path") {
		t.Errorf("Expected 'filename, not a path' error, got: %v", err)
	}
}

func TestRenameFile_NewNameContainsBackslash(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.RenameFile(context.Background(), "proj-1", "old.txt", "sub\\new.txt")
	if err == nil {
		t.Fatal("Expected error when new name contains backslash")
	}
}

func TestRenameFile_SourceNotFound(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.RenameFile(context.Background(), "proj-1", "nonexistent.txt", "new.txt")
	if err == nil {
		t.Fatal("Expected error for non-existent source")
	}
	if !strings.Contains(err.Error(), "source file not found") {
		t.Errorf("Expected 'source file not found', got: %v", err)
	}
}

func TestRenameFile_BlockedSourcePath(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	err := mgr.RenameFile(context.Background(), "proj-1", ".git/config", "new.txt")
	if err == nil {
		t.Fatal("Expected error for blocked source path")
	}
}

func TestRenameFile_BlockedTargetName(t *testing.T) {
	dir := t.TempDir()
	// Create a file inside a ".git" directory so the target is blocked
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "old"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(staticResolver{dir: dir})
	// Source is blocked
	err := mgr.RenameFile(context.Background(), "proj-1", ".git/old", "new")
	if err == nil {
		t.Fatal("Expected error for blocked path")
	}
}

// --- extractPatchFiles ---

func TestExtractPatchFiles_DiffGitHeader(t *testing.T) {
	patch := `diff --git a/foo.go b/foo.go
index abc123..def456 100644
--- a/foo.go
+++ b/foo.go
@@ -1,3 +1,3 @@
 package main
-var x = 1
+var x = 2
`
	files, err := extractPatchFiles(patch)
	if err != nil {
		t.Fatalf("extractPatchFiles: %v", err)
	}
	if len(files) < 1 {
		t.Fatal("Expected at least 1 file")
	}
	found := false
	for _, f := range files {
		if f == "foo.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected foo.go in files, got: %v", files)
	}
}

func TestExtractPatchFiles_MultipleDiffs(t *testing.T) {
	patch := `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1 +1 @@
-old
+new
diff --git a/b.go b/b.go
--- a/b.go
+++ b/b.go
@@ -1 +1 @@
-old
+new
`
	files, err := extractPatchFiles(patch)
	if err != nil {
		t.Fatalf("extractPatchFiles: %v", err)
	}
	if len(files) < 2 {
		t.Fatalf("Expected at least 2 files, got %d: %v", len(files), files)
	}
}

func TestExtractPatchFiles_NewFile(t *testing.T) {
	patch := `diff --git a/new.go b/new.go
new file mode 100644
--- /dev/null
+++ b/new.go
@@ -0,0 +1 @@
+package new
`
	files, err := extractPatchFiles(patch)
	if err != nil {
		t.Fatalf("extractPatchFiles: %v", err)
	}
	found := false
	for _, f := range files {
		if f == "new.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected new.go in files, got: %v", files)
	}
}

func TestExtractPatchFiles_Empty(t *testing.T) {
	_, err := extractPatchFiles("")
	if err == nil {
		t.Fatal("Expected error for empty patch")
	}
	if !strings.Contains(err.Error(), "no files found") {
		t.Errorf("Expected 'no files found' error, got: %v", err)
	}
}

func TestExtractPatchFiles_NoPatchHeaders(t *testing.T) {
	_, err := extractPatchFiles("this is not a patch\nno patch headers here\n")
	if err == nil {
		t.Fatal("Expected error for invalid patch")
	}
}

// --- ApplyPatch ---

func TestApplyPatch_EmptyPatch(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.ApplyPatch(context.Background(), "proj-1", "")
	if err == nil {
		t.Fatal("Expected error for empty patch")
	}
	if !strings.Contains(err.Error(), "patch is required") {
		t.Errorf("Expected 'patch is required', got: %v", err)
	}
}

func TestApplyPatch_WhitespacePatch(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.ApplyPatch(context.Background(), "proj-1", "   ")
	if err == nil {
		t.Fatal("Expected error for whitespace patch")
	}
}

func TestApplyPatch_SensitiveFile(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	patch := `diff --git a/.env b/.env
--- /dev/null
+++ b/.env
@@ -0,0 +1 @@
+SECRET=abc
`
	_, err := mgr.ApplyPatch(context.Background(), "proj-1", patch)
	if err == nil {
		t.Fatal("Expected error for sensitive file (.env)")
	}
	if !strings.Contains(err.Error(), "sensitive") {
		t.Errorf("Expected 'sensitive' error, got: %v", err)
	}
}

func TestApplyPatch_SensitivePasswordFile(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	patch := `diff --git a/password.txt b/password.txt
--- /dev/null
+++ b/password.txt
@@ -0,0 +1 @@
+pass123
`
	_, err := mgr.ApplyPatch(context.Background(), "proj-1", patch)
	if err == nil {
		t.Fatal("Expected error for sensitive password file")
	}
}

func TestApplyPatch_BlockedGitPath(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	patch := `diff --git a/.git/config b/.git/config
--- a/.git/config
+++ b/.git/config
@@ -1 +1 @@
-old
+new
`
	_, err := mgr.ApplyPatch(context.Background(), "proj-1", patch)
	if err == nil {
		t.Fatal("Expected error for blocked .git path")
	}
}

func TestApplyPatch_InvalidPatchFormat(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(staticResolver{dir: dir})
	_, err := mgr.ApplyPatch(context.Background(), "proj-1", "not a valid patch at all")
	if err == nil {
		t.Fatal("Expected error for invalid patch format")
	}
}

// --- helper functions ---

func TestSafeJoin(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		rel     string
		wantErr bool
	}{
		{"normal file", "/base", "file.txt", false},
		{"subdirectory", "/base", "sub/file.txt", false},
		{"empty rel becomes dot", "/base", "", false},
		{"dot path", "/base", ".", false},
		{"absolute path", "/base", "/etc/passwd", true},
		{"path traversal", "/base", "../escape", true},
		{"double traversal", "/base", "../../escape", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := safeJoin(tt.base, tt.rel)
			if (err != nil) != tt.wantErr {
				t.Errorf("safeJoin(%s, %s) error = %v, wantErr %v", tt.base, tt.rel, err, tt.wantErr)
			}
			if err == nil && result == "" {
				t.Error("Expected non-empty result")
			}
		})
	}
}

func TestIsBlockedPath(t *testing.T) {
	tests := []struct {
		path    string
		blocked bool
	}{
		{"/project/.git/config", true},
		{"/project/.git", true},
		{"/project/.git/hooks/pre-commit", true},
		{"/project/src/main.go", false},
		{"/project/.gitignore", false},
		{"/project/.github/workflows/ci.yml", false},
		{"/project/sub/.git/config", true},
	}

	for _, tt := range tests {
		result := isBlockedPath(tt.path)
		if result != tt.blocked {
			t.Errorf("isBlockedPath(%s) = %v, want %v", tt.path, result, tt.blocked)
		}
	}
}

func TestDepthFromPath(t *testing.T) {
	tests := []struct {
		path  string
		depth int
	}{
		{".", 0},
		{"", 0},
		{"file.txt", 1},
		{"sub/file.txt", 2},
		{"a/b/c/d.txt", 4},
		{"a/b/c", 3},
	}

	for _, tt := range tests {
		result := depthFromPath(tt.path)
		if result != tt.depth {
			t.Errorf("depthFromPath(%s) = %d, want %d", tt.path, result, tt.depth)
		}
	}
}

func TestReadWithLimit(t *testing.T) {
	tests := []struct {
		name    string
		content string
		limit   int64
		wantErr bool
	}{
		{"within limit", "hello", 100, false},
		{"exactly at limit", "hello", 5, false},
		{"exceeds limit", "hello world this is too long", 5, true},
		{"empty content", "", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.content)
			result, err := readWithLimit(r, tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("readWithLimit error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && result != tt.content {
				t.Errorf("readWithLimit = %s, want %s", result, tt.content)
			}
		})
	}
}

func TestResolveWorkDir(t *testing.T) {
	mgr := NewManager(staticResolver{dir: "/some/dir"})
	dir, err := mgr.resolveWorkDir("proj-1")
	if err != nil {
		t.Fatalf("resolveWorkDir: %v", err)
	}
	if dir == "" {
		t.Error("Expected non-empty workdir")
	}
}

func TestResolveWorkDir_NilResolver(t *testing.T) {
	mgr := NewManager(nil)
	_, err := mgr.resolveWorkDir("proj-1")
	if err == nil {
		t.Fatal("Expected error for nil resolver")
	}
}

func TestResolveWorkDir_EmptyResult(t *testing.T) {
	mgr := NewManager(emptyResolver{})
	_, err := mgr.resolveWorkDir("proj-1")
	if err == nil {
		t.Fatal("Expected error for empty result")
	}
}

// --- struct field tests ---

func TestFileResult_Fields(t *testing.T) {
	fr := FileResult{Path: "a.txt", Content: "data", Size: 4}
	if fr.Path != "a.txt" {
		t.Errorf("Expected path 'a.txt', got '%s'", fr.Path)
	}
	if fr.Content != "data" {
		t.Errorf("Expected content 'data', got '%s'", fr.Content)
	}
	if fr.Size != 4 {
		t.Errorf("Expected size 4, got %d", fr.Size)
	}
}

func TestTreeEntry_Fields(t *testing.T) {
	te := TreeEntry{Path: "sub/file.txt", Type: "file", Depth: 2}
	if te.Path != "sub/file.txt" {
		t.Errorf("Expected path 'sub/file.txt', got '%s'", te.Path)
	}
	if te.Type != "file" {
		t.Errorf("Expected type 'file', got '%s'", te.Type)
	}
	if te.Depth != 2 {
		t.Errorf("Expected depth 2, got %d", te.Depth)
	}
}

func TestSearchMatch_Fields(t *testing.T) {
	sm := SearchMatch{Path: "a.go", Line: 5, Text: "hello"}
	if sm.Path != "a.go" {
		t.Errorf("Expected path 'a.go', got '%s'", sm.Path)
	}
	if sm.Line != 5 {
		t.Errorf("Expected line 5, got %d", sm.Line)
	}
	if sm.Text != "hello" {
		t.Errorf("Expected text 'hello', got '%s'", sm.Text)
	}
}

func TestPatchResult_Fields(t *testing.T) {
	pr := PatchResult{Applied: true, Output: "ok"}
	if !pr.Applied {
		t.Error("Expected applied true")
	}
	if pr.Output != "ok" {
		t.Errorf("Expected output 'ok', got '%s'", pr.Output)
	}
}

func TestWriteResult_Fields(t *testing.T) {
	wr := WriteResult{Path: "x.txt", BytesWritten: 42}
	if wr.Path != "x.txt" {
		t.Errorf("Expected path 'x.txt', got '%s'", wr.Path)
	}
	if wr.BytesWritten != 42 {
		t.Errorf("Expected bytes written 42, got %d", wr.BytesWritten)
	}
}
