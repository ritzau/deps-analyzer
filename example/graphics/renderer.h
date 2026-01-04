#pragma once

#include <string>

namespace graphics {

class Renderer {
public:
  Renderer();
  ~Renderer();

  void Initialize();
  void DrawFrame();
  void SetResolution(int width, int height);
};

} // namespace graphics

