#include "audio/audio_engine.h"
#include "util/strings.h"
#include <iostream>

namespace audio {

AudioEngine::AudioEngine() {
    std::cout << "AudioEngine created\n";
}

AudioEngine::~AudioEngine() {
    std::cout << "AudioEngine destroyed\n";
}

void AudioEngine::Initialize() {
    std::cout << "AudioEngine initialized\n";
}

void AudioEngine::PlaySound(const std::string& filename) {
    std::string upper = util::ToUpper(filename);
    std::cout << "Playing sound: " << upper << "\n";
}

void AudioEngine::SetVolume(float volume) {
    std::cout << "Volume set to " << volume << "\n";
}

} // namespace audio
