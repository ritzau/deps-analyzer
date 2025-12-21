#ifndef UTIL_TIME_H_
#define UTIL_TIME_H_

#include <string>
#include <cstdint>

namespace util {

// Time utilities
int64_t GetCurrentTimeMillis();
int64_t GetCurrentTimeMicros();
std::string FormatTime(int64_t timestamp_millis);
void Sleep(int64_t millis);

}  // namespace util

#endif  // UTIL_TIME_H_
