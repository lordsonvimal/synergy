export function speak(text: string, speed = 1.0, voice?: string): void {
  if (!("speechSynthesis" in window)) {
    return;
  }

  window.speechSynthesis.cancel();

  const utterance = new SpeechSynthesisUtterance(text);
  utterance.rate = speed;

  if (voice) {
    const voices = window.speechSynthesis.getVoices();
    const match = voices.find((v) => v.name === voice);
    if (match) {
      utterance.voice = match;
    }
  }

  window.speechSynthesis.speak(utterance);
}

export function stopSpeaking(): void {
  if ("speechSynthesis" in window) {
    window.speechSynthesis.cancel();
  }
}

export function getVoices(): SpeechSynthesisVoice[] {
  if (!("speechSynthesis" in window)) {
    return [];
  }
  return window.speechSynthesis.getVoices();
}
