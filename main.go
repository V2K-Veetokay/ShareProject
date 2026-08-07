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

	http.HandleFunc("/upload", handleUpload)
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

func handleRequest(w http.ResponseWriter, r *http.Request) {
	requestPath := filepath.Clean(r.URL.Path)
	if strings.HasPrefix(requestPath, "/") {
		requestPath = requestPath[1:]
	}

	fullPath := filepath.Join(sharedDir, requestPath)
	absPath, err := filepath.Abs(fullPath)
	if err != nil || !strings.HasPrefix(absPath, sharedDir) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if info.IsDir() {
		handleListFiles(w, r, absPath, requestPath)
	} else {
		http.ServeFile(w, r, absPath)
	}
}

type Item struct {
	Name  string
	URL   string
	IsDir bool
	Size  string
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}

func handleListFiles(w http.ResponseWriter, r *http.Request, dirPath string, urlPath string) {
	uploaded := r.URL.Query().Get("uploaded") == "true"

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		http.Error(w, "Error reading directory", http.StatusInternalServerError)
		return
	}

	var items []Item
	for _, entry := range entries {
		itemName := entry.Name()
		itemURL := filepath.Join("/", urlPath, itemName)
		itemURL = strings.ReplaceAll(itemURL, "\\", "/")

		size := ""
		if !entry.IsDir() {
			info, _ := entry.Info()
			size = formatSize(info.Size())
		}

		items = append(items, Item{
			Name:  itemName,
			URL:   itemURL,
			IsDir: entry.IsDir(),
			Size:  size,
		})
	}

	parentURL := "/"
	if urlPath != "" && urlPath != "." {
		parentURL = "/" + filepath.Dir(urlPath)
		parentURL = strings.ReplaceAll(parentURL, "\\", "/")
	}

	tmpl := `
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>File Share</title>
		<style>
			* { margin: 0; padding: 0; box-sizing: border-box; }

			:root {
				--nord0: #2e3440;
				--nord1: #3b4252;
				--nord2: #434c5e;
				--nord3: #4c566a;
				--nord4: #d8dee9;
				--nord5: #e5e9f0;
				--nord6: #eceff4;
				--nord7: #8fbcbb;
				--nord8: #88c0d0;
				--nord9: #81a1c1;
				--nord10: #5e81ac;
				--nord11: #bf616a;
				--nord12: #d08770;
				--nord13: #ebcb8b;
				--nord14: #a3be8c;
				--nord15: #b48ead;
			}

			body {
				font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
				background: var(--nord0);
				min-height: 100vh;
				padding: 20px;
				color: var(--nord4);
			}

			.container {
				max-width: 900px;
				margin: 0 auto;
				background: var(--nord1);
				border-radius: 12px;
				box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3);
				overflow: hidden;
			}

			.header {
				background: linear-gradient(135deg, var(--nord9) 0%, var(--nord10) 100%);
				color: var(--nord6);
				padding: 30px;
				text-align: center;
			}

			.header h1 {
				font-size: 2.5em;
				margin-bottom: 10px;
			}

			.header p {
				opacity: 0.9;
				font-size: 0.95em;
			}

			.breadcrumb {
				padding: 15px 30px;
				background: var(--nord2);
				border-bottom: 1px solid var(--nord3);
				font-size: 0.95em;
			}

			.breadcrumb a {
				color: var(--nord8);
				text-decoration: none;
				font-weight: 500;
				transition: color 0.2s;
			}

			.breadcrumb a:hover {
				color: var(--nord7);
				text-decoration: underline;
			}

			.content {
				padding: 30px;
			}

			.success-message {
				background: rgba(163, 190, 140, 0.2);
				border: 1px solid var(--nord14);
				color: var(--nord14);
				padding: 12px 15px;
				border-radius: 6px;
				margin-bottom: 20px;
				font-weight: 500;
			}

			.upload-section {
				background: var(--nord2);
				border: 2px dashed var(--nord8);
				border-radius: 8px;
				padding: 30px;
				text-align: center;
				margin-bottom: 30px;
				transition: all 0.3s ease;
			}

			.upload-section:hover {
				border-color: var(--nord7);
				background: var(--nord3);
			}

			.upload-section h3 {
				margin-bottom: 15px;
				color: var(--nord6);
				font-size: 1.2em;
			}

			.upload-form {
				display: flex;
				gap: 10px;
				justify-content: center;
				flex-wrap: wrap;
			}

			.upload-form input[type="file"] {
				padding: 8px 12px;
				border: 1px solid var(--nord8);
				border-radius: 6px;
				cursor: pointer;
				background: var(--nord1);
				color: var(--nord4);
			}

			.upload-form input[type="file"]::file-selector-button {
				background: var(--nord9);
				color: var(--nord6);
				border: none;
				padding: 8px 16px;
				border-radius: 4px;
				cursor: pointer;
				font-weight: 600;
				transition: background 0.2s;
			}

			.upload-form input[type="file"]::file-selector-button:hover {
				background: var(--nord10);
			}

			.upload-form button {
				background: linear-gradient(135deg, var(--nord8) 0%, var(--nord9) 100%);
				color: var(--nord0);
				border: none;
				padding: 10px 25px;
				border-radius: 6px;
				cursor: pointer;
				font-weight: 600;
				transition: transform 0.2s, box-shadow 0.2s;
			}

			.upload-form button:hover {
				transform: translateY(-2px);
				box-shadow: 0 5px 15px rgba(136, 192, 208, 0.3);
			}

			.items-section h2 {
				margin-bottom: 20px;
				color: var(--nord6);
				font-size: 1.3em;
			}

			.items-list {
				list-style: none;
			}

			.item {
				display: flex;
				align-items: center;
				justify-content: space-between;
				padding: 15px;
				border: 1px solid var(--nord3);
				border-radius: 8px;
				margin-bottom: 10px;
				transition: all 0.3s ease;
				background: var(--nord2);
			}

			.item:hover {
				background: var(--nord3);
				border-color: var(--nord8);
				transform: translateX(5px);
				box-shadow: 0 3px 10px rgba(136, 192, 208, 0.2);
			}

			.item-left {
				display: flex;
				align-items: center;
				gap: 15px;
				flex: 1;
				min-width: 0;
			}

			.item-icon {
				font-size: 1.8em;
				min-width: 30px;
			}

			.item-name {
				flex: 1;
				min-width: 0;
				word-break: break-word;
			}

			.item-name a {
				color: var(--nord8);
				text-decoration: none;
				font-weight: 500;
				font-size: 1.05em;
				transition: color 0.2s;
			}

			.item-name a:hover {
				text-decoration: underline;
				color: var(--nord7);
			}

			.item-size {
				color: var(--nord4);
				font-size: 0.9em;
				min-width: 70px;
				text-align: right;
				opacity: 0.7;
			}

			.empty-state {
				text-align: center;
				padding: 40px 20px;
				color: var(--nord3);
			}

			.parent-item {
				background: rgba(191, 97, 106, 0.1);
				border-color: var(--nord11);
			}

			.parent-item:hover {
				background: rgba(191, 97, 106, 0.2);
				border-color: var(--nord12);
			}

			.parent-item a {
				color: var(--nord12);
				font-weight: 600;
			}

			@media (max-width: 600px) {
				.header h1 { font-size: 1.8em; }
				.item { flex-direction: column; align-items: flex-start; }
				.item-size { text-align: left; margin-top: 10px; }
				.upload-form { flex-direction: column; }
				.upload-form input, .upload-form button { width: 100%; }
			}
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1>📁 File Share</h1>
				<p>Share files easily on your local network</p>
			</div>

			<div class="breadcrumb">
				📍 <a href="/">Home</a>{{if .CurrentPath}} / {{.CurrentPath}}{{end}}
			</div>

			<div class="content">
				{{if .Success}}
				<div class="success-message">✅ File uploaded successfully!</div>
				{{end}}

				<div class="upload-section">
					<h3>📤 Upload a File</h3>
					<form class="upload-form" action="/upload" method="POST" enctype="multipart/form-data">
						<input type="file" name="file" required>
						<button type="submit">Upload</button>
					</form>
				</div>

				<div class="items-section">
					<h2>📂 Contents</h2>
					{{if .Items}}
					<ul class="items-list">
						{{if .ShowParent}}
						<li class="item parent-item">
							<div class="item-left">
								<div class="item-icon">⬆️</div>
								<div class="item-name">
									<a href="{{.ParentURL}}">Parent Directory</a>
								</div>
							</div>
						</li>
						{{end}}
						{{range .Items}}
						<li class="item">
							<div class="item-left">
								<div class="item-icon">
									{{if .IsDir}}📁{{else}}📄{{end}}
								</div>
								<div class="item-name">
									{{if .IsDir}}
									<a href="{{.URL}}">{{.Name}}</a>
									{{else}}
									<a href="{{.URL}}" download>{{.Name}}</a>
									{{end}}
								</div>
							</div>
							{{if .Size}}
							<div class="item-size">{{.Size}}</div>
							{{end}}
						</li>
						{{end}}
					</ul>
					{{else}}
					<div class="empty-state">
						<p>📭 This directory is empty</p>
					</div>
					{{end}}
				</div>
			</div>
		</div>
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
