#include "formatter/formatter.h"
#include <fmt/color.h>
#include <fmt/core.h>
#include <vector>

namespace formatter {

std::string format_colored(const std::string &text, const std::string &color) {
  // Use fmt library to format with colors
  if (color == "red") {
    return fmt::format(fg(fmt::color::red), "{}", text);
  } else if (color == "green") {
    return fmt::format(fg(fmt::color::green), "{}", text);
  } else if (color == "blue") {
    return fmt::format(fg(fmt::color::blue), "{}", text);
  }
  return fmt::format("{}", text);
}

std::string format_list(const std::vector<std::string> &items) {
  if (items.empty()) {
    return "[]";
  }

  std::string result = "[";
  for (size_t i = 0; i < items.size(); ++i) {
    result += fmt::format("'{}'", items[i]);
    if (i < items.size() - 1) {
      result += ", ";
    }
  }
  result += "]";
  return result;
}

} // namespace formatter
