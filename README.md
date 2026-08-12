# Simple File Share Server

A lightweight file sharing server built with Go that allows you to share files over your local network. It features a web-based interface for browsing, uploading, and downloading files and directories.

## Features

- **Web-based file browser** with a clean, modern interface
- **File upload** via drag-and-drop or form submission
- **Directory download** as ZIP archives
- **Local network access** with automatic IP detection
- **Path traversal protection** to prevent unauthorized access
- **Responsive design** that works on mobile and desktop devices
- **Nord theme**配色 for visual consistency

## Technical Overview

The server is built with Go's standard library, providing:
- HTTP file serving with path validation
- ZIP archive generation for directory downloads
- Template-based HTML rendering
- Network interface detection for local IP discovery

### Security Considerations

The application implements basic path traversal protection by verifying that all requested paths remain within the shared directory. However, this is not a production-grade security solution and should only be used in trusted environments.

## Installation

### Prerequisites

- Go 1.16 or higher
- A terminal/command prompt

### Build Instructions

1. Clone or download the source code
2. Navigate to the project directory
3. Run the following command:

```bash
go build -o fileshare main.go
```

This creates an executable named `fileshare` (or `fileshare.exe` on Windows).

## Usage

### Basic Usage

Run the server with default settings (port 8080, current directory):

```bash
./fileshare
```

### Command Line Options

```bash
./fileshare -port 9000 -dir /path/to/share
```

Available flags:
- `-port`: Port number to listen on (default: 8080)
- `-dir`: Directory to share (default: current directory)

### Example Commands

```bash
# Share current directory on port 8080
./fileshare

# Share /home/user/documents on port 9000
./fileshare -port 9000 -dir /home/user/documents

# Windows example
fileshare.exe -port 8080 -dir "C:\Shared Files"
```

## Accessing the Server

After starting the server, you will see output similar to:

```
------------------------------------------------
🚀 File Server started on port 8080
📂 Sharing directory: /home/user/documents
🌐 Local Access: http://localhost:8080
📡 Network Access: http://192.168.1.100:8080
   (Share the Network Access URL with others on your WiFi)
------------------------------------------------
```

- **Local access**: Open http://localhost:8080 in your browser
- **Network access**: Share the network URL with other devices on your LAN

## Web Interface Features

### File Operations
- **View files**: Click on any file to download it
- **Browse directories**: Click on folders to navigate
- **Upload files**: Use the upload form to add files to the shared directory
- **Download directories**: Click the ZIP button next to folders to download them as archives

### Navigation
- **Breadcrumb trail**: Shows current path with links to parent directories
- **File size display**: Shows file sizes in appropriate units (B, KB, MB, GB)
- **Mobile responsive**: Interface adapts to different screen sizes

## Default Configuration

- **Max upload size**: 100 MB
- **Default port**: 8080
- **Default directory**: Current working directory (`.`)

## Project Structure

```
ShareProject/
├── main.go           # Main application code
└── README.md         # This documentation
```

## Limitations

- **No authentication**: Anyone on the network with access can upload and download files
- **Single directory**: Only shares one directory at a time
- **No user management**: All users have the same permissions
- **Local network only**: Not designed for internet exposure

## Troubleshooting

### Server won't start
- Check if the port is already in use: `lsof -i :8080` (macOS/Linux) or `netstat -ano | findstr :8080` (Windows)
- Ensure the directory path is valid and accessible
- Check firewall settings to allow incoming connections

### Can't access from another device
- Verify both devices are on the same network
- Try accessing via the network IP instead of localhost
- Check firewall settings on the server machine
- Ensure the network is set to private/trusted (not public)

### Upload fails
- Verify the upload size is within the 100 MB limit
- Check file permissions on the shared directory
- Ensure sufficient disk space is available

## License

This project is provided as-is for personal and educational use.

## Contributing

Contributions are welcome. Please feel free to submit issues or pull requests.

## Support

For issues or questions, please check the troubleshooting section or review the source code for detailed implementation details.
