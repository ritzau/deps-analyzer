#ifndef UTIL_MATH_H_
#define UTIL_MATH_H_

#include <cstdint>
#include <string>

namespace util {

// Math utilities
int64_t Gcd(int64_t a, int64_t b);
int64_t Lcm(int64_t a, int64_t b);
bool IsPrime(int64_t n);
double Clamp(double value, double min, double max);

// Note: This creates a dependency on strings (for demonstration)
std::string NumberToString(int64_t n);

} // namespace util

#endif // UTIL_MATH_H_
