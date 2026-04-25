package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
)

var (
	port        = flag.Int("p", 13377, "Port to serve on")
	dir         = flag.String("dir", ".", "Directory to serve")
	readOnly    = flag.Bool("read-only", false, "Enable read-only mode")
	uploadOnly  = flag.Bool("upload-only", false, "Enable upload-only mode")
	authCreds   = flag.String("auth", "", "Enable basic auth in the format user:pass")
	runAsUser   = flag.String("user", "", "Drop privileges to this UNIX user")
	logFilePath = flag.String("log", "", "Path to log file")
)

// defaultPort is the "quick share" port. When the server is launched on this
// port we enable UploadedOnly mode, which scopes browsing/serving/deleting
// to the uploads directory only.
const defaultPort = 13377

//go:embed templates/* static/*
var embeddedFiles embed.FS

func main() {
	flag.Parse()

	if *logFilePath != "" {
		f, err := os.OpenFile(*logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Unable to open log file: %v", err)
		}
		log.SetOutput(f)
	}

	chosenDir, err := filepath.Abs(filepath.Clean(*dir))
	if err != nil {
		log.Fatalf("Failed to resolve dir %s: %v", *dir, err)
	}

	uploadDir := filepath.Join(chosenDir, "uploads")
	if err := os.MkdirAll(uploadDir, 0750); err != nil {
		log.Fatalf("Failed to create upload dir %s: %v", uploadDir, err)
	}

	// If we're going to drop privileges, make sure the target user actually
	// owns both the chosen dir and the uploads dir, otherwise the post-drop
	// process won't be able to write into them. This was the bug behind
	// "uploads silently fail on first install" — /var/kutta was created
	// as root by the Makefile and the service user couldn't write to it.
	if *runAsUser != "" && os.Geteuid() == 0 {
		u, err := user.Lookup(*runAsUser)
		if err != nil {
			log.Fatalf("Failed to find user %s: %v", *runAsUser, err)
		}

		uid, err := strconv.Atoi(u.Uid)
		if err != nil {
			log.Fatalf("Invalid uid for %s: %v", *runAsUser, err)
		}
		gid, err := strconv.Atoi(u.Gid)
		if err != nil {
			log.Fatalf("Invalid gid for %s: %v", *runAsUser, err)
		}

		if err := os.Chown(chosenDir, uid, gid); err != nil {
			log.Printf("Warning: failed to chown %s to %s: %v", chosenDir, *runAsUser, err)
		}
		if err := os.Chown(uploadDir, uid, gid); err != nil {
			log.Fatalf("Failed to chown upload dir %s to %s: %v", uploadDir, *runAsUser, err)
		}
	}

	if *runAsUser != "" {
		u, err := user.Lookup(*runAsUser)
		if err != nil {
			log.Fatalf("Failed to find user: %v", err)
		}
		if err := dropPrivileges(u.Uid, u.Gid); err != nil {
			log.Fatalf("Failed to drop privileges: %v", err)
		}
	}

	h := &kuttaHandler{
		Dir:          chosenDir,
		UploadDir:    uploadDir,
		ReadOnly:     *readOnly,
		UploadOnly:   *uploadOnly,
		AuthEnabled:  *authCreds != "",
		AuthCreds:    *authCreds,
		FS:           embeddedFiles,
		UploadedOnly: *port == defaultPort,
		Port:         *port,
	}
	h.RegisterRoutes()

	staticFS, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		log.Fatalf("Failed to load static FS: %v", err)
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Kuttañ serving payloads on http://localhost%s (dir: %s, uploads: %s)", addr, chosenDir, uploadDir)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
