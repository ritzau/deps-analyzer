#pragma once

#include <string>

namespace formatter {

// Format a message with color using the external fmt library
std::string format_colored(const std::string &text, const std::string &color);

// Format a list of items
std::string format_list(const std::vector<std::string> &items);

} // namespace formatter

