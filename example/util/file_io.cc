#include "util/file_io.h"
#include <dirent.h>
#include <fstream>
#include <sstream>
#include <sys/stat.h>

namespace util {

bool ReadFile(const std::string &path, std::string *content) {
  std::ifstream file(path);
  if (!file.is_open()) {
    return false;
  }

  std::stringstream buffer;
  buffer << file.rdbuf();
  *content = buffer.str();
  return true;
}

bool WriteFile(const std::string &path, const std::string &content) {
  std::ofstream file(path);
  if (!file.is_open()) {
    return false;
  }

  file << content;
  return true;
}

bool FileExists(const std::string &path) {
  struct stat buffer;
  return (stat(path.c_str(), &buffer) == 0);
}

std::vector<std::string> ListDirectory(const std::string &path) {
  std::vector<std::string> entries;
  DIR *dir = opendir(path.c_str());
  if (!dir) {
    return entries;
  }

  struct dirent *entry;
  while ((entry = readdir(dir)) != nullptr) {
    std::string name = entry->d_name;
    if (name != "." && name != "..") {
      entries.push_back(name);
    }
  }

  closedir(dir);
  return entries;
}

} // namespace util

