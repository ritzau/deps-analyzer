#ifndef CYCLE_DEMO_FILE_B_H
#define CYCLE_DEMO_FILE_B_H

// Forward declare from file_a to create dependency
void functionA();

void functionFromB() {
  functionA(); // This creates a dependency on file_a
}

#endif
