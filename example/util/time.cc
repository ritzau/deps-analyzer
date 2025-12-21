#include "util/time.h"
#include <chrono>
#include <thread>
#include <ctime>
#include <iomanip>
#include <sstream>

namespace util {

int64_t GetCurrentTimeMillis() {
  auto now = std::chrono::system_clock::now();
  auto duration = now.time_since_epoch();
  return std::chrono::duration_cast<std::chrono::milliseconds>(duration).count();
}

int64_t GetCurrentTimeMicros() {
  auto now = std::chrono::system_clock::now();
  auto duration = now.time_since_epoch();
  return std::chrono::duration_cast<std::chrono::microseconds>(duration).count();
}

std::string FormatTime(int64_t timestamp_millis) {
  auto time_point = std::chrono::system_clock::time_point(
      std::chrono::milliseconds(timestamp_millis));
  auto time_t = std::chrono::system_clock::to_time_t(time_point);
  
  std::stringstream ss;
  ss << std::put_time(std::localtime(&time_t), "%Y-%m-%d %H:%M:%S");
  return ss.str();
}

void Sleep(int64_t millis) {
  std::this_thread::sleep_for(std::chrono::milliseconds(millis));
}

}  // namespace util
