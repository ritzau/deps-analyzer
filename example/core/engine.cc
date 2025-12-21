#include "core/engine.h"
#include <iostream>

namespace core {

Engine::Engine(const std::string& name) 
    : name_(util::ToUpper(name)), 
      start_time_(0),
      initialized_(false) {
}

Engine::~Engine() {
  if (initialized_) {
    Shutdown();
  }
}

void Engine::Initialize() {
  if (initialized_) {
    std::cout << "Engine already initialized" << std::endl;
    return;
  }
  
  start_time_ = util::GetCurrentTimeMillis();
  initialized_ = true;
  std::cout << "Engine '" << name_ << "' initialized" << std::endl;
}

void Engine::Shutdown() {
  if (!initialized_) {
    return;
  }
  
  initialized_ = false;
  std::cout << "Engine '" << name_ << "' shutdown after " 
            << GetUptime() << "ms" << std::endl;
}

void Engine::Update() {
  if (!initialized_) {
    std::cout << "Engine not initialized!" << std::endl;
    return;
  }
  // Update logic here
}

int64_t Engine::GetUptime() const {
  if (!initialized_) {
    return 0;
  }
  return util::GetCurrentTimeMillis() - start_time_;
}

}  // namespace core
