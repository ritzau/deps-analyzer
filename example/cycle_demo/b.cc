#include "cycle_demo/a.h"

void funcB() {
    funcA();  // Call function from a.h, which will pull in a.cc
}
