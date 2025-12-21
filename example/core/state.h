#ifndef CORE_STATE_H_
#define CORE_STATE_H_

#include <string>
#include <map>
#include "util/file_io.h"  // Cross-package dependency

namespace core {

// State manager - handles application state
class StateManager {
 public:
  StateManager();
  
  void SetValue(const std::string& key, const std::string& value);
  std::string GetValue(const std::string& key) const;
  bool HasKey(const std::string& key) const;
  
  bool SaveToFile(const std::string& path);
  bool LoadFromFile(const std::string& path);
  
 private:
  std::map<std::string, std::string> state_;
};

}  // namespace core

#endif  // CORE_STATE_H_
