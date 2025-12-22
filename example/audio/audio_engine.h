#pragma once

#include <string>

namespace audio {

class AudioEngine {
public:
    AudioEngine();
    ~AudioEngine();

    void Initialize();
    void PlaySound(const std::string& filename);
    void SetVolume(float volume);
};

} // namespace audio
