#pragma once

#include <map>
#include <string>

namespace config {

// Configuration manager using JSON for storage
class ConfigManager {
public:
  ConfigManager();

  // Load configuration from JSON string
  bool LoadFromJson(const std::string &json_str);

  // Get configuration as JSON string
  std::string ToJson() const;

  // Set/get configuration values
  void SetValue(const std::string &key, const std::string &value);
  std::string GetValue(const std::string &key,
                       const std::string &default_value = "") const;

  // Set/get integer values
  void SetInt(const std::string &key, int value);
  int GetInt(const std::string &key, int default_value = 0) const;

private:
  std::map<std::string, std::string> config_;
};

} // namespace config

