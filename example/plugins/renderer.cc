#include "plugins/renderer.h"
#include <iostream>

namespace plugins {

Renderer::Renderer() : engine_(nullptr), initialized_(false) {}

Renderer::~Renderer() {}

void Renderer::Initialize(core::Engine *engine) {
  engine_ = engine;
  initialized_ = true;
  std::cout << "Renderer initialized for engine: " << engine->GetName()
            << std::endl;
}

void Renderer::Render() {
  if (!initialized_) {
    std::cout << "Renderer not initialized!" << std::endl;
    return;
  }

  std::cout << "Rendering frame for: " << title_ << std::endl;
}

void Renderer::SetTitle(const std::string &title) {
  title_ = util::ToUpper(title); // Use util function
}

// Plugin interface implementation
extern "C" {
Renderer *CreateRenderer() { return new Renderer(); }

void DestroyRenderer(Renderer *renderer) { delete renderer; }
}

} // namespace plugins

