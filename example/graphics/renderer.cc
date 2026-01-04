#include "graphics/renderer.h"
#include <dlfcn.h> // System library for dynamic loading
#include <iostream>

namespace graphics {

Renderer::Renderer() { std::cout << "Renderer created\n"; }

Renderer::~Renderer() { std::cout << "Renderer destroyed\n"; }

void Renderer::Initialize() {
  std::cout << "Renderer initialized\n";
  // Use dlopen to demonstrate system library dependency
  void *handle = dlopen(nullptr, RTLD_LAZY);
  if (handle) {
    dlclose(handle);
  }
}

void Renderer::DrawFrame() { std::cout << "Drawing frame\n"; }

void Renderer::SetResolution(int width, int height) {
  std::cout << "Resolution set to " << width << "x" << height << "\n";
}

} // namespace graphics
