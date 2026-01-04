#include "util/strings.h"
#include <algorithm>
#include <sstream>

namespace util {

std::string ToUpper(const std::string &str) {
  std::string result = str;
  std::transform(result.begin(), result.end(), result.begin(), ::toupper);
  return result;
}

std::string ToLower(const std::string &str) {
  std::string result = str;
  std::transform(result.begin(), result.end(), result.begin(), ::tolower);
  return result;
}

std::vector<std::string> Split(const std::string &str, char delimiter) {
  std::vector<std::string> parts;
  std::stringstream ss(str);
  std::string part;
  while (std::getline(ss, part, delimiter)) {
    parts.push_back(part);
  }
  return parts;
}

std::string Join(const std::vector<std::string> &parts,
                 const std::string &separator) {
  if (parts.empty())
    return "";

  std::string result = parts[0];
  for (size_t i = 1; i < parts.size(); ++i) {
    result += separator + parts[i];
  }
  return result;
}

bool StartsWith(const std::string &str, const std::string &prefix) {
  return str.size() >= prefix.size() &&
         str.compare(0, prefix.size(), prefix) == 0;
}

bool EndsWith(const std::string &str, const std::string &suffix) {
  return str.size() >= suffix.size() &&
         str.compare(str.size() - suffix.size(), suffix.size(), suffix) == 0;
}

} // namespace util

