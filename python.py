import http.server
import socketserver
import os
import socket
from pathlib import Path

# Get the folder where this script is located
SHARE_FOLDER = os.path.dirname(os.path.abspath(__file__))
PORT = 8000

def get_local_ip():
    """Get the local machine IP address"""
    try:
        # Connect to a public DNS server (doesn't actually send data)
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.connect(("8.8.8.8", 80))
        ip = s.getsockname()[0]
        s.close()
        return ip
    except Exception:
        return "127.0.0.1"

class MyHTTPRequestHandler(http.server.SimpleHTTPRequestHandler):
    def translate_path(self, path):
        # Serve files from SHARE_FOLDER instead of current directory
        path = super().translate_path(path)
        relpath = os.path.relpath(path, os.getcwd())
        return os.path.join(SHARE_FOLDER, relpath)

os.chdir(SHARE_FOLDER)

local_ip = get_local_ip()

with socketserver.TCPServer(("", PORT), MyHTTPRequestHandler) as httpd:
    print(f"Sharing folder: {SHARE_FOLDER}")
    print(f"Access it at: http://localhost:{PORT}")
    print(f"Or from another machine: http://{local_ip}:{PORT}")
    print("Press Ctrl+C to stop")
    httpd.serve_forever()
