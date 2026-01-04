#include "cycle_demo/b.h"

void funcA() {
  funcB(); // Call function from b.h, which will pull in b.cc
}

