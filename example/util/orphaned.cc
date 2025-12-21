#include <iostream>

namespace util {

// This file is intentionally NOT included in any BUILD target
// The analyzer should detect this as an uncovered file

void OrphanedFunction() {
  std::cout << "I'm an orphan - not in any BUILD target!" << std::endl;
}

}  // namespace util
