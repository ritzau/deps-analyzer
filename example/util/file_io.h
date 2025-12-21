#ifndef UTIL_FILE_IO_H_
#define UTIL_FILE_IO_H_

#include <string>
#include <vector>

namespace util {

// File I/O utilities
bool ReadFile(const std::string& path, std::string* content);
bool WriteFile(const std::string& path, const std::string& content);
bool FileExists(const std::string& path);
std::vector<std::string> ListDirectory(const std::string& path);

}  // namespace util

#endif  // UTIL_FILE_IO_H_
