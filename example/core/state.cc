#include "core/state.h"
#include <sstream>

namespace core {

StateManager::StateManager() {}

void StateManager::SetValue(const std::string& key, const std::string& value) {
  state_[key] = value;
}

std::string StateManager::GetValue(const std::string& key) const {
  auto it = state_.find(key);
  if (it != state_.end()) {
    return it->second;
  }
  return "";
}

bool StateManager::HasKey(const std::string& key) const {
  return state_.find(key) != state_.end();
}

bool StateManager::SaveToFile(const std::string& path) {
  std::stringstream content;
  for (const auto& pair : state_) {
    content << pair.first << "=" << pair.second << "\n";
  }
  return util::WriteFile(path, content.str());
}

bool StateManager::LoadFromFile(const std::string& path) {
  std::string content;
  if (!util::ReadFile(path, &content)) {
    return false;
  }
  
  // Simple parsing (key=value per line)
  std::stringstream ss(content);
  std::string line;
  state_.clear();
  
  while (std::getline(ss, line)) {
    size_t pos = line.find('=');
    if (pos != std::string::npos) {
      std::string key = line.substr(0, pos);
      std::string value = line.substr(pos + 1);
      state_[key] = value;
    }
  }
  
  return true;
}

}  // namespace core
