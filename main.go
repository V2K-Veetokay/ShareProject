package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
)

func main() {
	// Configuration flags
	port := flag.String("port", "8080", "Port to listen on")
	dir := flag.String("dir", ".", "Directory to share")
	flag.Parse()

	// Resolve the absolute path of the directory to share
	absDir, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("Invalid directory: %v", err)
	}

	// Create a file server
	// http.Dir(absDir) serves the files from that directory
	fs := http.FileServer(http.Dir(absDir))
	http.Handle("/", fs)

	// Determine the local IP address
	localIP, err := getLocalIP()
	if err != nil {
		log.Printf("Could not determine local IP: %v", err)
		log.Println("Make sure to use your local IP manually (e.g., 192.168.x.x)")
	}

	// Start the server
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

// getLocalIP finds the first non-loopback IPv4 address
func getLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// Skip loopback, down, or internal interfaces
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

			// Check if it's IPv4 and not a loopback
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}

			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("no valid IP found")
}
