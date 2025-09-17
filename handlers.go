package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// to track uploaded files
var uploadedFiles = make(map[string]bool)

type FileEntry struct {
	Name    string
	Size    string
	ModTime string
	Icon    string
	IsDir   bool
}

type ClipboardEntry struct {
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

var clipboard []ClipboardEntry

type kuttaHandler struct {
	Dir          string
	ReadOnly     bool
	UploadOnly   bool
	AuthEnabled  bool
	AuthCreds    string
	FS           embed.FS
	UploadedOnly bool // show only uploaded files (default port mode)
}

func (h *kuttaHandler) RegisterRoutes() {
	http.HandleFunc("/", h.indexHandler)
	http.HandleFunc("/upload", h.uploadHandler)
	http.HandleFunc("/delete", h.deleteHandler)
	http.HandleFunc("/bulkdelete", h.bulkDeleteHandler)
	http.HandleFunc("/files/", h.fileServeHandler)
	http.HandleFunc("/clipboard", h.clipboardHandler)
	http.HandleFunc("/clipboard/export", h.clipboardExportHandler)
	http.HandleFunc("/clipboard/clear", h.clipboardClearHandler)
}

func (h *kuttaHandler) indexHandler(w http.ResponseWriter, r *http.Request) {
	if h.UploadOnly {
		http.Error(w, "Access denied in upload-only mode", http.StatusForbidden)
		return
	}

	tmpl, err := template.ParseFS(h.FS, "templates/index.html")
	if err != nil {
		http.Error(w, "Template error", 500)
		return
	}

	// current relative path
	relPath := strings.TrimPrefix(r.URL.Path, "/")
	if relPath == "" {
		relPath = "."
	}
	curDir := filepath.Join(h.Dir, relPath)

	entries, err := os.ReadDir(curDir)
	if err != nil {
		http.Error(w, "Failed to list directory", 500)
		return
	}

	// build file list
	var files []FileEntry
	query := strings.ToLower(r.URL.Query().Get("q"))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		name := entry.Name()
		if query != "" && !strings.Contains(strings.ToLower(name), query) {
			continue
		}
		fullPath := filepath.Join(curDir, name)

		// skip existing files if UploadedOnly is true
		if h.UploadedOnly && !uploadedFiles[fullPath] {
			continue
		}

		files = append(files, FileEntry{
			Name:    name,
			Size:    humanReadableSize(info.Size()),
			ModTime: info.ModTime().Format(time.RFC822),
			Icon:    iconForFile(name, entry.IsDir()),
			IsDir:   entry.IsDir(),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	// determine parent path for ".."
	var parentPath string
	if relPath != "." {
		parentPath = filepath.Dir(relPath)
		if parentPath == "." {
			parentPath = ""
		}
	}

	// render web template
	tmpl.Execute(w, map[string]interface{}{
		"Files":      files,
		"ReadOnly":   h.ReadOnly,
		"UploadOnly": h.UploadOnly,
		"Dir":        curDir,
		"RelPath":    relPath,
		"ParentPath": parentPath,
	})
}

func (h *kuttaHandler) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if h.ReadOnly {
		http.Error(w, "Uploads disabled in read-only mode", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	curDir := h.Dir
	if curDir == "" {
		curDir = "."
	}

	if r.Method == http.MethodPost {
		r.ParseMultipartForm(50 << 20)
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		outPath := filepath.Join(curDir, header.Filename)

		// prevent overwrite
		if _, err := os.Stat(outPath); err == nil {
			ext := filepath.Ext(header.Filename)
			name := strings.TrimSuffix(header.Filename, ext)
			outPath = filepath.Join(curDir, fmt.Sprintf("%s_%d%s", name, time.Now().UnixNano(), ext))
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			http.Error(w, "Failed to save file", 500)
			return
		}
		defer outFile.Close()
		io.Copy(outFile, file)

		// mark as uploaded
		uploadedFiles[outPath] = true

		log.Printf("Uploaded via POST: %s", outPath)
		http.Redirect(w, r, r.Referer(), http.StatusSeeOther)

	} else if r.Method == http.MethodPut {
		filename := filepath.Base(r.URL.Path)
		if filename == "" {
			http.Error(w, "Missing filename in URL", http.StatusBadRequest)
			return
		}

		outPath := filepath.Join(curDir, filename)
		if _, err := os.Stat(outPath); err == nil {
			ext := filepath.Ext(filename)
			name := strings.TrimSuffix(filename, ext)
			outPath = filepath.Join(curDir, fmt.Sprintf("%s_%d%s", name, time.Now().UnixNano(), ext))
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			http.Error(w, "Failed to save file", 500)
			return
		}
		defer outFile.Close()

		if _, err := io.Copy(outFile, r.Body); err != nil {
			http.Error(w, "Failed to write file", 500)
			return
		}

		// mark as uploaded
		uploadedFiles[outPath] = true

		log.Printf("Uploaded via PUT: %s", outPath)
		w.WriteHeader(http.StatusCreated)
	}
}

func (h *kuttaHandler) fileServeHandler(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimPrefix(r.URL.Path, "/files/")
	file := filepath.Join(h.Dir, relPath)
	http.ServeFile(w, r, file)
}

func (h *kuttaHandler) deleteHandler(w http.ResponseWriter, r *http.Request) {
	if h.ReadOnly {
		http.Error(w, "Delete not allowed in read-only mode", http.StatusForbidden)
		return
	}
	file := r.URL.Query().Get("file")
	path := filepath.Join(h.Dir, file)

	// only allow delete if file was uploaded
	if !uploadedFiles[path] {
		http.Error(w, "Cannot delete existing file", http.StatusForbidden)
		return
	}

	if err := os.Remove(path); err != nil {
		http.Error(w, "Failed to delete file", 500)
		return
	}
	delete(uploadedFiles, path) // clean-up
	log.Printf("Deleted: %s", file)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *kuttaHandler) bulkDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if h.ReadOnly {
		http.Error(w, "Delete not allowed in read-only mode", http.StatusForbidden)
		return
	}
	r.ParseForm()
	for _, file := range r.Form["files"] {
		path := filepath.Join(h.Dir, file)
		if uploadedFiles[path] {
			os.Remove(path)
			delete(uploadedFiles, path)
			log.Printf("Bulk deleted: %s", file)
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// clipboard handlers
func (h *kuttaHandler) clipboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		text := r.Form.Get("text")
		if text != "" {
			clipboard = append(clipboard, ClipboardEntry{Text: text, Timestamp: time.Now()})
		}
		w.WriteHeader(http.StatusCreated)
		return
	} else if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(clipboard)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *kuttaHandler) clipboardExportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Disposition", "attachment; filename=clipboard.json")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clipboard)
}

func (h *kuttaHandler) clipboardClearHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		clipboard = nil
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// helpers
func humanReadableSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
}

func iconForFile(name string, isDir bool) string {
	if isDir {
		return "📂"
	}
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".exe"), strings.HasSuffix(lower, ".bin"):
		return "💻"
	case strings.HasSuffix(lower, ".dll"):
		return "🧩"
	case strings.HasSuffix(lower, ".sys"):
		return "🔒"
	case strings.HasSuffix(lower, ".iso"), strings.HasSuffix(lower, ".img"):
		return "📀"
	case strings.HasSuffix(lower, ".txt"), strings.HasSuffix(lower, ".md"), strings.HasSuffix(lower, ".log"):
		return "📄"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"), strings.HasSuffix(lower, ".png"), strings.HasSuffix(lower, ".gif"):
		return "🖼️"
	case strings.HasSuffix(lower, ".zip"), strings.HasSuffix(lower, ".tar"), strings.HasSuffix(lower, ".gz"), strings.HasSuffix(lower, ".rar"):
		return "📦"
	case strings.HasSuffix(lower, ".go"), strings.HasSuffix(lower, ".c"), strings.HasSuffix(lower, ".cpp"), strings.HasSuffix(lower, ".py"), strings.HasSuffix(lower, ".js"):
		return "⚙️"
	default:
		return "📄"
	}
}
