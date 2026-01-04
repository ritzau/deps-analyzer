#include "config/config.h"
#include "core/engine.h"
#include "core/state.h"
#include "formatter/formatter.h"
#include "graphics/renderer.h"
#include "util/file_io.h"
#include "util/strings.h"
#include "util/time.h"
#include <dlfcn.h>
#include <iostream>
#include <vector>

int main(int argc, char **argv) {
  std::cout << "=== Test Application ===" << std::endl;

  // Use core functionality
  core::Engine engine("TestEngine");
  engine.Initialize();

  // Use util functionality
  std::string test = "hello world";
  std::cout << "Uppercase: " << util::ToUpper(test) << std::endl;

  std::cout << "Current time: "
            << util::FormatTime(util::GetCurrentTimeMillis()) << std::endl;

  // Use state manager
  core::StateManager state;
  state.SetValue("version", "1.0");
  state.SetValue("name", "test_app");

  std::cout << "State version: " << state.GetValue("version") << std::endl;

  // Use external formatter library
  std::vector<std::string> features = {"engine", "graphics", "plugins",
                                       "formatting"};
  std::cout << "\nFeatures: " << formatter::format_list(features) << std::endl;
  std::cout << formatter::format_colored("Status: OK", "green") << std::endl;

  // Use JSON config manager (nlohmann/json via http_archive)
  config::ConfigManager cfg;
  cfg.SetValue("app_name", "TestApp");
  cfg.SetInt("max_connections", 100);
  cfg.SetValue("debug_mode", "true");

  std::cout << "\nConfiguration (JSON):\n" << cfg.ToJson() << std::endl;

  // Load from JSON
  std::string json_cfg =
      R"({"width": 1920, "height": 1080, "fullscreen": false})";
  if (cfg.LoadFromJson(json_cfg)) {
    std::cout << "Loaded display config: " << cfg.GetInt("width") << "x"
              << cfg.GetInt("height") << std::endl;
  }

  // Use graphics library (dynamically linked)
  graphics::Renderer renderer;
  renderer.Initialize();
  renderer.SetResolution(1920, 1080);
  renderer.DrawFrame();

  // Try to load plugin (demonstration of plugin loading)
  std::cout << "\nAttempting to load plugin..." << std::endl;

  // In a real scenario, you would load the .so file here
  // void* handle = dlopen("./plugins/librenderer_plugin.so", RTLD_LAZY);
  // if (handle) {
  //   auto create_fn = (plugins::Renderer*(*)())dlsym(handle,
  //   "CreateRenderer"); if (create_fn) {
  //     auto* renderer = create_fn();
  //     renderer->Initialize(&engine);
  //     renderer->SetTitle("My App");
  //     renderer->Render();
  //   }
  //   dlclose(handle);
  // }

  std::cout << "\nApplication running for: " << engine.GetUptime() << "ms"
            << std::endl;

  engine.Shutdown();

  return 0;
}

