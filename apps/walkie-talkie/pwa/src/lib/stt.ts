export interface STTCallbacks {
  onInterim: (text: string) => void;
  onFinal: (text: string) => void;
  onError: (error: string) => void;
  onEnd?: () => void;
}

interface SpeechRecognitionEvent {
  results: SpeechRecognitionResultList;
  resultIndex: number;
}

interface SpeechRecognitionErrorEvent {
  error: string;
}

interface SpeechRecognition extends EventTarget {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  start: () => void;
  stop: () => void;
  abort: () => void;
  onresult: ((event: SpeechRecognitionEvent) => void) | null;
  onerror: ((event: SpeechRecognitionErrorEvent) => void) | null;
  onend: (() => void) | null;
  onaudiostart: (() => void) | null;
}

declare global {
  interface Window {
    webkitSpeechRecognition: new () => SpeechRecognition;
    SpeechRecognition: new () => SpeechRecognition;
  }
}

export function createSTT(callbacks: STTCallbacks): {
  start: () => void;
  stop: () => void;
  supported: boolean;
} | null {
  const SpeechRecognition =
    window.webkitSpeechRecognition || window.SpeechRecognition;

  if (!SpeechRecognition) {
    return null;
  }

  const recognition = new SpeechRecognition();
  recognition.continuous = true;
  recognition.interimResults = true;
  recognition.lang = "en-IN";

  let hadResult = false;
  let isRunning = false;
  let shouldRestart = false;

  recognition.onresult = (event: SpeechRecognitionEvent) => {
    let interim = "";
    let final = "";

    for (let i = event.resultIndex; i < event.results.length; i++) {
      const result = event.results[i];
      if (!result) {
        continue;
      }
      const transcript = result[0]?.transcript ?? "";
      if (result.isFinal) {
        final += transcript;
      } else {
        interim += transcript;
      }
    }

    if (final) {
      hadResult = true;
      callbacks.onFinal(final);
    } else if (interim) {
      callbacks.onInterim(interim);
    }
  };

  recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
    callbacks.onError(event.error);
  };

  recognition.onend = () => {
    isRunning = false;
    if (shouldRestart) {
      hadResult = false;
      isRunning = true;
      recognition.start();
      return;
    }
    if (!hadResult) {
      callbacks.onEnd?.();
    }
    hadResult = false;
  };

  return {
    start: () => {
      if (isRunning) {
        return;
      }
      hadResult = false;
      shouldRestart = true;
      isRunning = true;
      recognition.start();
    },
    stop: () => {
      shouldRestart = false;
      if (isRunning) {
        recognition.stop();
      }
    },
    supported: true
  };
}
