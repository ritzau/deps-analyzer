#ifndef CORE_ENGINE_H_
#define CORE_ENGINE_H_

#include "util/strings.h" // Cross-package dependency
#include "util/time.h"    // Cross-package dependency
#include <string>

namespace core {

class Engine {
public:
  Engine(const std::string &name);
  ~Engine();

  void Initialize();
  void Shutdown();
  void Update();

  std::string GetName() const { return name_; }
  int64_t GetUptime() const;

private:
  std::string name_;
  int64_t start_time_;
  bool initialized_;
};

} // namespace core

#endif // CORE_ENGINE_H_

