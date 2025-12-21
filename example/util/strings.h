#ifndef UTIL_STRINGS_H_
#define UTIL_STRINGS_H_

#include <string>
#include <vector>

namespace util {

// String manipulation utilities
std::string ToUpper(const std::string& str);
std::string ToLower(const std::string& str);
std::vector<std::string> Split(const std::string& str, char delimiter);
std::string Join(const std::vector<std::string>& parts, const std::string& separator);
bool StartsWith(const std::string& str, const std::string& prefix);
bool EndsWith(const std::string& str, const std::string& suffix);

}  // namespace util

#endif  // UTIL_STRINGS_H_
