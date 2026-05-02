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
  let lastInterim = "";

  const getTranscript = (result: SpeechRecognitionResult): string =>
    result[0]?.transcript ?? "";

  const processResults = (event: SpeechRecognitionEvent): { interim: string; final: string } => {
    let interim = "";
    let final = "";
    for (let i = event.resultIndex; i < event.results.length; i++) {
      const result = event.results[i];
      if (!result) continue;
      if (result.isFinal) { final += getTranscript(result); } else { interim += getTranscript(result); }
    }
    return { interim, final };
  };

  const emitResults = (interim: string, final: string): void => {
    if (final) {
      hadResult = true;
      lastInterim = "";
      callbacks.onFinal(final);
    } else if (interim) {
      lastInterim = interim;
      callbacks.onInterim(interim);
    }
  };

  recognition.onresult = (event: SpeechRecognitionEvent) => {
    const { interim, final } = processResults(event);
    emitResults(interim, final);
  };

  recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
    callbacks.onError(event.error);
  };

  recognition.onend = () => {
    isRunning = false;
    if (shouldRestart) {
      if (lastInterim) {
        callbacks.onFinal(lastInterim);
        lastInterim = "";
      }
      hadResult = false;
      isRunning = true;
      recognition.start();
      return;
    }
    if (!hadResult) {
      callbacks.onEnd?.();
    }
    hadResult = false;
    lastInterim = "";
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
