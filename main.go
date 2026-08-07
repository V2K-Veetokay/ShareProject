package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	maxUploadSize = int64(100) * 1024 * 1024 // 100 MB default
	sharedDir     = "."
)

func main() {
	port := flag.String("port", "8080", "Port to listen on")
	dir := flag.String("dir", ".", "Directory to share")
	flag.Parse()

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("Invalid directory: %v", err)
	}
	sharedDir = absDir

	// Handle uploads
	http.HandleFunc("/upload", handleUpload)

	// Handle all requests (files, folders, and directory listing)
	http.HandleFunc("/", handleRequest)

	localIP, err := getLocalIP()
	if err != nil {
		log.Printf("Could not determine local IP: %v", err)
		log.Println("Make sure to use your local IP manually (e.g., 192.168.x.x)")
	}

	addr := ":" + *port
	fmt.Println("------------------------------------------------")
	fmt.Printf("🚀 File Server started on port %s\n", *port)
	fmt.Printf("📂 Sharing directory: %s\n", absDir)

	if localIP != "" {
		fmt.Printf("🌐 Local Access: http://localhost:%s\n", *port)
		fmt.Printf("📡 Network Access: http://%s:%s\n", localIP, *port)
		fmt.Println("   (Share the Network Access URL with others on your WiFi)")
	} else {
		fmt.Printf("📡 Access: http://localhost:%s\n", *port)
	}
	fmt.Println("------------------------------------------------")
	fmt.Println("Press Ctrl+C to stop")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// handleRequest handles file downloads, folder navigation, and directory listing
func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Sanitize the path
	requestPath := filepath.Clean(r.URL.Path)
	if strings.HasPrefix(requestPath, "/") {
		requestPath = requestPath[1:]
	}

	fullPath := filepath.Join(sharedDir, requestPath)

	// Security check: ensure the path is within sharedDir
	absPath, err := filepath.Abs(fullPath)
	if err != nil || !strings.HasPrefix(absPath, sharedDir) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// If it's a file, serve it for download
	if info.IsDir() {
		// If it's a directory, show directory listing
		handleListFiles(w, r, absPath, requestPath)
	} else {
		// Serve file for download
		http.ServeFile(w, r, absPath)
	}
}

// handleListFiles lists files and folders with an upload form
func handleListFiles(w http.ResponseWriter, r *http.Request, dirPath string, urlPath string) {
	uploaded := r.URL.Query().Get("uploaded") == "true"

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		http.Error(w, "Error reading directory", http.StatusInternalServerError)
		return
	}

	type Item struct {
		Name string
		URL  string
		IsDir bool
	}

	var items []Item
	for _, entry := range entries {
		itemName := entry.Name()
		itemURL := filepath.Join("/", urlPath, itemName)
		itemURL = strings.ReplaceAll(itemURL, "\\", "/") // Windows compatibility

		items = append(items, Item{
			Name:  itemName,
			URL:   itemURL,
			IsDir: entry.IsDir(),
		})
	}

	// Determine parent directory URL
	parentURL := "/"
	if urlPath != "" && urlPath != "." {
		parentURL = "/" + filepath.Dir(urlPath)
		parentURL = strings.ReplaceAll(parentURL, "\\", "/")
	}

	tmpl := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>File Share</title>
		<style>
			body { font-family: sans-serif; padding: 20px; max-width: 800px; margin: 0 auto; }
			.breadcrumb { margin-bottom: 20px; }
			.breadcrumb a { color: #007bff; text-decoration: none; }
			.breadcrumb a:hover { text-decoration: underline; }
			.item-list { list-style: none; padding: 0; }
			.item-list li { margin: 8px 0; padding: 8px; border: 1px solid #ddd; border-radius: 3px; }
			.item-list a { text-decoration: none; color: #007bff; }
			.item-list a:hover { text-decoration: underline; }
			.folder { font-weight: bold; }
			.upload-form { border: 1px solid #ccc; padding: 20px; margin-bottom: 20px; border-radius: 5px; background: #f9f9f9; }
			.success { color: green; margin-bottom: 10px; font-weight: bold; }
		</style>
	</head>
	<body>
		<h1>📂 File Share</h1>

		{{if .Success}}
		<div class="success">✅ File uploaded successfully!</div>
		{{end}}

		<div class="breadcrumb">
			📍 <a href="/">Home</a>{{if .CurrentPath}} / {{.CurrentPath}}{{end}}
		</div>

		<div class="upload-form">
			<h3>📤 Upload File</h3>
			<form action="/upload" method="POST" enctype="multipart/form-data">
				<input type="file" name="file" required>
				<button type="submit">Upload</button>
			</form>
		</div>

		<h2>📂 Contents</h2>
		{{if .Items}}
		<ul class="item-list">
			{{if .ShowParent}}
			<li><a href="{{.ParentURL}}">📁 .. (Parent Directory)</a></li>
			{{end}}
			{{range .Items}}
			<li>
				{{if .IsDir}}
				<span class="folder">📁 <a href="{{.URL}}">{{.Name}}</a></span>
				{{else}}
				<span>📄 <a href="{{.URL}}" download>{{.Name}}</a></span>
				{{end}}
			</li>
			{{end}}
		</ul>
		{{else}}
		<p>This directory is empty.</p>
		{{end}}
	</body>
	</html>
	`

	t, err := template.New("filelist").Parse(tmpl)
	if err != nil {
		log.Fatalf("Template parse error: %v", err)
	}

	data := struct {
		Items       []Item
		Success     bool
		CurrentPath string
		ParentURL   string
		ShowParent  bool
	}{
		Items:       items,
		Success:     uploaded,
		CurrentPath: urlPath,
		ParentURL:   parentURL,
		ShowParent:  urlPath != "" && urlPath != ".",
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

// handleUpload processes the file upload
func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(maxUploadSize)

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		log.Printf("Error retrieving file: %v", err)
		return
	}
	defer file.Close()

	filename := filepath.Base(handler.Filename)
	if !filepath.IsLocal(filename) {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	dst, err := os.Create(filepath.Join(sharedDir, filename))
	if err != nil {
		http.Error(w, "Error creating file", http.StatusInternalServerError)
		log.Printf("Error creating file: %v", err)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		log.Printf("Error saving file: %v", err)
		return
	}

	http.Redirect(w, r, "/?uploaded=true", http.StatusSeeOther)
}

// getLocalIP finds the first non-loopback IPv4 address
func getLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}

			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("no valid IP found")
}
