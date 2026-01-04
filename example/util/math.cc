#include "util/math.h"
#include "util/strings.h" // Internal dependency within util package
#include <cmath>
#include <sstream>

namespace util {

int64_t Gcd(int64_t a, int64_t b) {
  while (b != 0) {
    int64_t temp = b;
    b = a % b;
    a = temp;
  }
  return a;
}

int64_t Lcm(int64_t a, int64_t b) { return (a / Gcd(a, b)) * b; }

bool IsPrime(int64_t n) {
  if (n <= 1)
    return false;
  if (n <= 3)
    return true;
  if (n % 2 == 0 || n % 3 == 0)
    return false;

  for (int64_t i = 5; i * i <= n; i += 6) {
    if (n % i == 0 || n % (i + 2) == 0) {
      return false;
    }
  }
  return true;
}

double Clamp(double value, double min, double max) {
  if (value < min)
    return min;
  if (value > max)
    return max;
  return value;
}

std::string NumberToString(int64_t n) {
  std::stringstream ss;
  ss << n;
  std::string result = ss.str();
  // Use ToUpper just to create a dependency
  return ToUpper(result);
}

} // namespace util

