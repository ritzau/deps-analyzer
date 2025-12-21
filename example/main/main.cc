#include <iostream>
#include <dlfcn.h>
#include "core/engine.h"
#include "core/state.h"
#include "util/strings.h"
#include "util/time.h"
#include "util/file_io.h"

int main(int argc, char** argv) {
  std::cout << "=== Test Application ===" << std::endl;
  
  // Use core functionality
  core::Engine engine("TestEngine");
  engine.Initialize();
  
  // Use util functionality
  std::string test = "hello world";
  std::cout << "Uppercase: " << util::ToUpper(test) << std::endl;
  
  std::cout << "Current time: " 
            << util::FormatTime(util::GetCurrentTimeMillis()) 
            << std::endl;
  
  // Use state manager
  core::StateManager state;
  state.SetValue("version", "1.0");
  state.SetValue("name", "test_app");
  
  std::cout << "State version: " << state.GetValue("version") << std::endl;
  
  // Try to load plugin (demonstration of plugin loading)
  std::cout << "\nAttempting to load plugin..." << std::endl;
  
  // In a real scenario, you would load the .so file here
  // void* handle = dlopen("./plugins/librenderer_plugin.so", RTLD_LAZY);
  // if (handle) {
  //   auto create_fn = (plugins::Renderer*(*)())dlsym(handle, "CreateRenderer");
  //   if (create_fn) {
  //     auto* renderer = create_fn();
  //     renderer->Initialize(&engine);
  //     renderer->SetTitle("My App");
  //     renderer->Render();
  //   }
  //   dlclose(handle);
  // }
  
  std::cout << "\nApplication running for: " 
            << engine.GetUptime() << "ms" << std::endl;
  
  engine.Shutdown();
  
  return 0;
}
