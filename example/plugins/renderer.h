#ifndef PLUGINS_RENDERER_H_
#define PLUGINS_RENDERER_H_

#include <string>
#include "core/engine.h"  // Plugin depends on core
#include "util/strings.h"  // And util

namespace plugins {

class Renderer {
 public:
  Renderer();
  ~Renderer();
  
  void Initialize(core::Engine* engine);
  void Render();
  void SetTitle(const std::string& title);
  
 private:
  core::Engine* engine_;
  std::string title_;
  bool initialized_;
};

// Plugin interface
extern "C" {
  Renderer* CreateRenderer();
  void DestroyRenderer(Renderer* renderer);
}

}  // namespace plugins

#endif  // PLUGINS_RENDERER_H_
