export interface STTCallbacks {
  onTranscript: (text: string) => void;
  onError: (error: string) => void;
  onEnd?: () => void;
  onStateChange?: (state: "listening" | "speech-detected" | "not-listening") => void;
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
  onaudioend: (() => void) | null;
  onspeechstart: (() => void) | null;
  onspeechend: (() => void) | null;
}

declare global {
  interface Window {
    webkitSpeechRecognition: new () => SpeechRecognition;
    SpeechRecognition: new () => SpeechRecognition;
  }
}

const STARTUP_TIMEOUT_MS = 5000;
const NO_RESULT_TIMEOUT_MS = 10000;
const RESTART_DELAY_MS = 100;
const MAX_RESTART_FAILURES = 3;

export function createSTT(callbacks: STTCallbacks): {
  start: () => void;
  stop: () => void;
  getTranscript: () => string;
  isActive: () => boolean;
  supported: boolean;
} | null {
  const SpeechRecognition =
    window.webkitSpeechRecognition || window.SpeechRecognition;

  if (!SpeechRecognition) {
    return null;
  }

  let recognition: SpeechRecognition | null = null;
  let active = false;
  let audioStarted = false;
  let speechDetected = false;
  let resultReceived = false;
  let segments: string[] = [];
  let currentInterim = "";
  let startupTimer: ReturnType<typeof setTimeout> | undefined;
  let noResultTimer: ReturnType<typeof setTimeout> | undefined;
  let restartTimer: ReturnType<typeof setTimeout> | undefined;
  let restartFailures = 0;

  function clearTimers(): void {
    clearTimeout(startupTimer);
    clearTimeout(noResultTimer);
    clearTimeout(restartTimer);
  }

  function getFullTranscript(): string {
    const parts = currentInterim
      ? [...segments, currentInterim]
      : segments;
    return parts.join(" ");
  }

  function emitTranscript(): void {
    callbacks.onTranscript(getFullTranscript());
  }

  function startRecognition(): void {
    recognition = createRecognition();
    try {
      recognition.start();
      if (!audioStarted) {
        startupTimer = setTimeout(() => {
          if (!active || audioStarted) return;
          callbacks.onError("startup-timeout");
        }, STARTUP_TIMEOUT_MS);
      }
    } catch (err) {
      active = false;
      clearTimers();
      const message =
        err instanceof Error ? err.message : "Failed to start recognition";
      callbacks.onError(message);
    }
  }

  function createRecognition(): SpeechRecognition {
    const rec = new SpeechRecognition();
    rec.continuous = false;
    rec.interimResults = true;
    rec.lang = "en-US";

    rec.onaudiostart = () => {
      audioStarted = true;
      restartFailures = 0;
      clearTimeout(startupTimer);
      callbacks.onStateChange?.("listening");
      noResultTimer = setTimeout(() => {
        if (!active || resultReceived) return;
        const reason = speechDetected ? "no-transcription" : "no-speech-detected";
        callbacks.onError(reason);
      }, NO_RESULT_TIMEOUT_MS);
    };

    rec.onspeechstart = () => {
      speechDetected = true;
      callbacks.onStateChange?.("speech-detected");
    };

    rec.onresult = (event: SpeechRecognitionEvent) => {
      resultReceived = true;
      clearTimeout(noResultTimer);
      // With continuous=false, there's only ever one result
      const result = event.results[0];
      if (!result?.[0]) return;
      if (result.isFinal) {
        const text = result[0].transcript.trim();
        if (text) {
          segments.push(text);
          currentInterim = "";
        }
      } else {
        currentInterim = result[0].transcript;
      }
      emitTranscript();
    };

    rec.onerror = (event: SpeechRecognitionErrorEvent) => {
      if (event.error === "aborted" && !active) return;
      if (event.error === "no-speech" && active) return;
      callbacks.onError(event.error);
    };

    rec.onend = () => {
      if (!active) return;
      // Flush any remaining interim
      if (currentInterim) {
        const text = currentInterim.trim();
        if (text) segments.push(text);
        currentInterim = "";
      }
      // Auto-restart for next utterance
      restartTimer = setTimeout(() => {
        if (!active) return;
        if (restartFailures >= MAX_RESTART_FAILURES) {
          active = false;
          callbacks.onEnd?.();
          return;
        }
        restartFailures++;
        startRecognition();
      }, RESTART_DELAY_MS);
    };

    return rec;
  }

  return {
    start: () => {
      if (active) return;
      audioStarted = false;
      speechDetected = false;
      resultReceived = false;
      restartFailures = 0;
      segments = [];
      currentInterim = "";
      active = true;
      startRecognition();
    },
    stop: () => {
      if (!active) return;
      active = false;
      clearTimers();
      // Flush interim
      if (currentInterim) {
        const text = currentInterim.trim();
        if (text) segments.push(text);
        currentInterim = "";
      }
      try {
        recognition?.stop();
      } catch {
        // Already stopped
      }
      recognition = null;
    },
    getTranscript: () => getFullTranscript(),
    isActive: () => active,
    supported: true
  };
}
