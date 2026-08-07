// share_server.cpp (for Arch Linux)
#include <iostream>
#include <filesystem>
#include <fstream>
#include <regex>
#include <vector>
#include <map>
#include <cstring>
#include <algorithm>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <sys/stat.h>
#include <httplib.h>
#include <nlohmann/json.hpp>

using json = nlohmann::json;
namespace fs = std::filesystem;

const int PORT = 8000;
std::string SHARE_FOLDER;

// Get local IP address
std::string get_local_ip() {
    try {
        int sock = socket(AF_INET, SOCK_DGRAM, 0);
        if (sock == -1) return "127.0.0.1";

        struct sockaddr_in servaddr;
        servaddr.sin_addr.s_addr = inet_addr("8.8.8.8");
        servaddr.sin_family = AF_INET;
        servaddr.sin_port = htons(80);

        if (connect(sock, (struct sockaddr*)&servaddr, sizeof(servaddr)) < 0) {
            close(sock);
            return "127.0.0.1";
        }

        struct sockaddr_in name;
        socklen_t namelen = sizeof(name);
        getsockname(sock, (struct sockaddr*)&name, &namelen);
        close(sock);

        return inet_ntoa(name.sin_addr);
    } catch (...) {
        return "127.0.0.1";
    }
}

// URL decode function
std::string url_decode(const std::string& src) {
    std::string result;
    for (size_t i = 0; i < src.length(); ++i) {
        if (src[i] == '%' && i + 2 < src.length()) {
            int hex_value;
            std::sscanf(src.substr(i + 1, 2).c_str(), "%x", &hex_value);
            result += static_cast<char>(hex_value);
            i += 2;
        } else if (src[i] == '+') {
            result += ' ';
        } else {
            result += src[i];
        }
    }
    return result;
}

// Format file size (bytes to human readable)
std::string format_file_size(size_t bytes) {
    if (bytes == 0) return "-";

    const char* units[] = {"B", "KB", "MB", "GB"};
    double size = static_cast<double>(bytes);
    int unit_index = 0;

    while (size >= 1024 && unit_index < 3) {
        size /= 1024;
        unit_index++;
    }

    char buffer[32];
    std::snprintf(buffer, sizeof(buffer), "%.2f %s", size, units[unit_index]);
    return std::string(buffer);
}

// Check if path is safe (no directory traversal)
bool is_path_safe(const std::string& path) {
    try {
        std::string abs_path = fs::absolute(path).string();
        std::string abs_share = fs::absolute(SHARE_FOLDER).string();
        return abs_path.compare(0, abs_share.length(), abs_share) == 0;
    } catch (...) {
        return false;
    }
}

// API endpoint: List files
void handle_list_api(const httplib::Request& req, httplib::Response& res) {
    std::string folder_path = url_decode(req.path.substr(9)); // Remove "/api/list"

    if (folder_path.empty() || folder_path == "/") {
        folder_path = SHARE_FOLDER;
    } else {
        folder_path = SHARE_FOLDER + "/" + folder_path;
    }

    if (!is_path_safe(folder_path)) {
        res.set_header("Content-Type", "application/json");
        res.status = 403;
        res.set_content("[]", "application/json");
        return;
    }

    if (!fs::is_directory(folder_path)) {
        res.set_header("Content-Type", "application/json");
        res.status = 404;
        res.set_content("[]", "application/json");
        return;
    }

    json items = json::array();
    std::vector<std::string> entries;

    try {
        for (const auto& entry : fs::directory_iterator(folder_path)) {
            entries.push_back(entry.path().filename().string());
        }
        std::sort(entries.begin(), entries.end());

        for (const auto& item : entries) {
            std::string item_path = folder_path + "/" + item;
            bool is_dir = fs::is_directory(item_path);
            size_t file_size = is_dir ? 0 : fs::file_size(item_path);

            std::string rel_path = fs::relative(item_path, SHARE_FOLDER).string();
            std::replace(rel_path.begin(), rel_path.end(), '\\', '/');

            json item_obj;
            item_obj["name"] = item;
            item_obj["isDir"] = is_dir;
            item_obj["size"] = file_size;
            item_obj["path"] = "/" + rel_path;

            items.push_back(item_obj);
        }
    } catch (...) {
        // Permission denied
    }

    res.set_header("Content-Type", "application/json");
    res.set_content(items.dump(), "application/json");
}

// API endpoint: Upload files
void handle_upload_api(const httplib::Request& req, httplib::Response& res) {
    std::string upload_path = url_decode(req.path.substr(11)); // Remove "/api/upload"

    std::string target_folder;
    if (upload_path.empty() || upload_path == "/") {
        target_folder = SHARE_FOLDER;
    } else {
        target_folder = SHARE_FOLDER + "/" + upload_path;
    }

    if (!is_path_safe(target_folder)) {
        res.status = 403;
        res.set_content(R"({"success":false,"error":"Access denied"})", "application/json");
        return;
    }

    if (!fs::is_directory(target_folder)) {
        res.status = 404;
        res.set_content(R"({"success":false,"error":"Folder not found"})", "application/json");
        return;
    }

    try {
        json uploaded_files = json::array();

        // Parse multipart form data
        std::string content_type = req.get_header_value("Content-Type");
        std::regex boundary_regex(R"(boundary=([^\s;]+))");
        std::smatch match;

        if (!std::regex_search(content_type, match, boundary_regex)) {
            res.status = 400;
            res.set_content(R"({"success":false,"error":"No boundary found"})", "application/json");
            return;
        }

        std::string boundary = "--" + std::string(match[1]);
        std::string body = req.body;
        size_t pos = 0;

        while ((pos = body.find(boundary, pos)) != std::string::npos) {
            pos += boundary.length();

            if (body.substr(pos, 2) == "--") break; // End boundary

            // Skip CRLF
            if (body.substr(pos, 2) == "\r\n") pos += 2;
            else if (body[pos] == '\n') pos += 1;

            // Find headers
            size_t header_end = body.find("\r\n\r\n", pos);
            if (header_end == std::string::npos) {
                header_end = body.find("\n\n", pos);
                if (header_end == std::string::npos) continue;
            }

            std::string headers = body.substr(pos, header_end - pos);

            // Extract filename
            std::regex filename_regex("filename=\"([^\"]+)\"");
            if (!std::regex_search(headers, match, filename_regex)) continue;

            std::string filename = match[1];
            filename = fs::path(filename).filename().string();
            if (filename.empty()) continue;

            // Get file content
            size_t content_start = header_end + 4;
            size_t content_end = body.find(boundary, content_start);
            if (content_end == std::string::npos) content_end = body.length();

            // Remove trailing CRLF/LF before boundary
            while (content_end > content_start &&
                   (body[content_end - 1] == '\n' || body[content_end - 1] == '\r')) {
                content_end--;
            }

            std::string file_content = body.substr(content_start, content_end - content_start);

            // Save file
            std::string file_path = target_folder + "/" + filename;
            std::ofstream file(file_path, std::ios::binary);
            file.write(file_content.c_str(), file_content.length());
            file.close();

            json file_obj;
            file_obj["name"] = filename;
            file_obj["size"] = file_content.length();
            uploaded_files.push_back(file_obj);
        }

        json response;
        response["success"] = true;
        response["files"] = uploaded_files;

        res.set_header("Content-Type", "application/json");
        res.set_content(response.dump(), "application/json");

    } catch (const std::exception& e) {
        json error_response;
        error_response["success"] = false;
        error_response["error"] = e.what();

        res.status = 500;
        res.set_header("Content-Type", "application/json");
        res.set_content(error_response.dump(), "application/json");
    }
}

// Get MIME type
std::string get_mime_type(const std::string& filename) {
    static const std::map<std::string, std::string> mime_types = {
        {".html", "text/html"},
        {".css", "text/css"},
        {".js", "application/javascript"},
        {".json", "application/json"},
        {".png", "image/png"},
        {".jpg", "image/jpeg"},
        {".jpeg", "image/jpeg"},
        {".gif", "image/gif"},
        {".svg", "image/svg+xml"},
        {".pdf", "application/pdf"},
        {".txt", "text/plain"},
        {".mp4", "video/mp4"},
        {".mp3", "audio/mpeg"},
    };

    std::string ext = filename.substr(filename.find_last_of("."));
    std::transform(ext.begin(), ext.end(), ext.begin(), ::tolower);

    auto it = mime_types.find(ext);
    return it != mime_types.end() ? it->second : "application/octet-stream";
}

// Load HTML from file
std::string get_html() {
    try {
        std::ifstream file("index.html", std::ios::binary);
        if (file) {
            return std::string((std::istreambuf_iterator<char>(file)), std::istreambuf_iterator<char>());
        }
    } catch (...) {
    }
    // Fallback if file not found
    return "<h1>Error: index.html not found</h1>";
}

        int main() {
            // Get script folder
            SHARE_FOLDER = fs::current_path().string();

            // Create HTTP server
            httplib::Server svr;

            // Register API endpoints
            svr.Get("/api/list.*", handle_list_api);
            svr.Post("/api/upload.*", handle_upload_api);

            // Serve index.html for root
            svr.Get("/", [](const httplib::Request& req, httplib::Response& res) {
                res.set_header("Content-Type", "text/html");
                res.set_content(get_html(), "text/html");
            });

            svr.Get("/index.html", [](const httplib::Request& req, httplib::Response& res) {
                res.set_header("Content-Type", "text/html");
                res.set_content(get_html(), "text/html");
            });

            // Serve files
            svr.Get("/.*", [](const httplib::Request& req, httplib::Response& res) {
                std::string file_path = SHARE_FOLDER + req.path;

                if (!is_path_safe(file_path)) {
                    res.status = 403;
                    return;
                }

                if (fs::is_regular_file(file_path)) {
                    std::ifstream file(file_path, std::ios::binary);
                    if (file) {
                        std::string mime = get_mime_type(file_path);
                        res.set_header("Content-Type", mime);
                        res.set_header("Content-Disposition",
                            "attachment; filename=" + fs::path(file_path).filename().string());

                        std::string content((std::istreambuf_iterator<char>(file)),
                                           std::istreambuf_iterator<char>());
                        res.set_content(content, mime);
                        file.close();
                    } else {
                        res.status = 404;
                    }
                } else {
                    res.status = 404;
                }
            });

            std::string local_ip = get_local_ip();

            std::cout << "\n" << std::string(50, '=') << std::endl;
            std::cout << " FILE SHARE SERVER STARTED" << std::endl;
            std::cout << std::string(50, '=') << std::endl;
            std::cout << " Sharing folder: " << SHARE_FOLDER << std::endl;
            std::cout << " Local access: http://localhost:" << PORT << std::endl;
            std::cout << " Network access: http://" << local_ip << ":" << PORT << std::endl;
            std::cout << std::string(50, '=') << std::endl;
            std::cout << "Press Ctrl+C to stop\n" << std::endl;

            svr.listen("0.0.0.0", PORT);

            return 0;
        }
