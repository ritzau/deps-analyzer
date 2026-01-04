#ifndef CYCLE_DEMO_FILE_A_H
#define CYCLE_DEMO_FILE_A_H

// Forward declare from file_b to create dependency
void functionFromB();

void functionA() {
  functionFromB(); // This creates a dependency on file_b
}

#endif
