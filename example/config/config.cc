#include "config/config.h"
#include <nlohmann/json.hpp>
#include <stdexcept>

using json = nlohmann::json;

namespace config {

ConfigManager::ConfigManager() {}

bool ConfigManager::LoadFromJson(const std::string &json_str) {
  try {
    auto j = json::parse(json_str);
    config_.clear();

    for (auto &[key, value] : j.items()) {
      if (value.is_string()) {
        config_[key] = value.get<std::string>();
      } else if (value.is_number_integer()) {
        config_[key] = std::to_string(value.get<int>());
      } else if (value.is_boolean()) {
        config_[key] = value.get<bool>() ? "true" : "false";
      }
    }
    return true;
  } catch (const json::exception &e) {
    return false;
  }
}

std::string ConfigManager::ToJson() const {
  json j;
  for (const auto &[key, value] : config_) {
    // Try to parse as int first
    try {
      int int_val = std::stoi(value);
      j[key] = int_val;
    } catch (...) {
      // Otherwise treat as string
      j[key] = value;
    }
  }
  return j.dump(2); // Pretty print with 2-space indent
}

void ConfigManager::SetValue(const std::string &key, const std::string &value) {
  config_[key] = value;
}

std::string ConfigManager::GetValue(const std::string &key,
                                    const std::string &default_value) const {
  auto it = config_.find(key);
  if (it != config_.end()) {
    return it->second;
  }
  return default_value;
}

void ConfigManager::SetInt(const std::string &key, int value) {
  config_[key] = std::to_string(value);
}

int ConfigManager::GetInt(const std::string &key, int default_value) const {
  auto it = config_.find(key);
  if (it != config_.end()) {
    try {
      return std::stoi(it->second);
    } catch (...) {
      return default_value;
    }
  }
  return default_value;
}

} // namespace config

