import { Component, createSignal, onCleanup, Show, For } from "solid-js";
import { createSTT } from "../lib/stt.js";
import { addToast } from "../lib/toast.js";
import { MicIcon, StopIcon } from "./icons.js";

interface VoiceInputProps {
  onSend: (text: string) => void;
  onReviewChange?: (reviewing: boolean) => void;
}

type SttState = "idle" | "listening" | "speech-detected" | "error";

export const VoiceInput: Component<VoiceInputProps> = (props) => {
  const [recording, setRecording] = createSignal(false);
  const [reviewText, setReviewText] = createSignal("");
  const [reviewing, setReviewing] = createSignal(false);
  const [recordingTime, setRecordingTime] = createSignal(0);
  const [sttState, setSttState] = createSignal<SttState>("idle");
  let timerInterval: ReturnType<typeof setInterval> | undefined;

  const formatTime = (seconds: number): string => {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m}:${s.toString().padStart(2, "0")}`;
  };

  const startTimer = (): void => {
    setRecordingTime(0);
    timerInterval = setInterval(() => {
      setRecordingTime(t => t + 1);
    }, 1000);
  };

  const stopTimer = (): void => {
    if (timerInterval) {
      clearInterval(timerInterval);
      timerInterval = undefined;
    }
    setRecordingTime(0);
  };

  const stopRecordingOnError = (): void => {
    setSttState("error");
    setRecording(false);
    stopTimer();
    stt?.stop();
  };

  type ToastLevel = "error" | "warning" | "info";

  const sttErrors: Record<string, [string, ToastLevel]> = {
    "not-allowed": ["Microphone permission denied. Check browser settings.", "error"],
    "network": ["Internet required for speech recognition", "error"],
    "no-speech": ["No speech detected — speak closer to the mic", "warning"],
    "audio-capture": ["Microphone unavailable — check if another app is using it", "error"],
    "service-not-allowed": ["Speech service blocked. Try reloading the page.", "error"],
    "language-not-supported": ["Speech language not supported on this device", "error"],
    "startup-timeout": ["Microphone not responding — check permissions and try again", "error"],
    "no-transcription": ["Speech heard but not transcribed — check internet", "error"],
    "no-speech-detected": ["Mic active but no voice detected — speak louder", "warning"],
  };

  const stt = createSTT({
    onTranscript: () => {
      setSttState("speech-detected");
    },
    onError: (err) => {
      if (err === "aborted") return;
      const entry = sttErrors[err];
      const [message, level] = entry ?? [`Speech recognition failed: ${err}`, "error" as ToastLevel];
      addToast(message, level);
      stopRecordingOnError();
    },
    onEnd: () => {
      if (!recording()) return;
      setRecording(false);
      stopTimer();
      setSttState("idle");
      addToast("Speech recognition stopped — tap mic to retry", "warning");
    },
    onStateChange: (state) => {
      if (state === "listening") setSttState("listening");
      if (state === "speech-detected") setSttState("speech-detected");
    },
  });

  onCleanup(() => {
    if (recording()) {
      stt?.stop();
    }
    stopTimer();
  });

  const stopRecording = (): void => {
    stt?.stop();
    const finalText = stt?.getTranscript().trim() ?? "";
    setRecording(false);
    stopTimer();
    setSttState("idle");
    if (!finalText) {
      addToast("No speech captured — try speaking louder or closer to the mic", "warning");
      return;
    }
    setReviewText(finalText);
    setReviewing(true);
    props.onReviewChange?.(true);
  };

  const handleToggle = (): void => {
    if (!stt) {
      addToast("Speech recognition not supported in this browser", "error");
      return;
    }
    if (recording()) {
      stopRecording();
    } else {
      setSttState("listening");
      setRecording(true);
      startTimer();
      stt.start();
    }
  };

  const handleSend = (): void => {
    const text = reviewText().trim();
    if (text) {
      props.onSend(text);
    }
    setReviewText("");
    setReviewing(false);
    props.onReviewChange?.(false);
  };

  const handleCancel = (): void => {
    setReviewText("");
    setReviewing(false);
    props.onReviewChange?.(false);
  };

  const handleKeyDown = (e: KeyboardEvent): void => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    } else if (e.key === "Escape") {
      e.preventDefault();
      handleCancel();
    }
  };

  const barColor = (): string => {
    const state = sttState();
    if (state === "speech-detected") return "bg-success";
    if (state === "error") return "bg-error";
    return "bg-warning";
  };

  return (
    <Show
      when={reviewing()}
      fallback={
        <div class="flex items-center gap-2">
          <Show when={recording()}>
            <div class="flex items-center gap-0.5 h-6" aria-hidden="true">
              <For each={[0, 1, 2, 3, 4]}>
                {i => (
                  <span
                    class={`w-1 rounded-full animate-waveform-bar transition-colors ${barColor()}`}
                    style={{ "animation-delay": `${i * 0.15}s` }}
                  />
                )}
              </For>
            </div>
          </Show>
          <Show when={recording()}>
            <span class="text-xs text-ink-secondary font-mono tabular-nums">
              {formatTime(recordingTime())}
            </span>
          </Show>
          <button
            class={`flex items-center justify-center w-10 h-10 rounded-full border-none text-on-primary cursor-pointer transition-transform active:scale-95 ${
              recording()
                ? "bg-error animate-pulse-recording"
                : "bg-primary hover:bg-primary-hover"
            }`}
            onClick={handleToggle}
            aria-label={recording() ? "Stop recording" : "Start recording"}
            data-testid="mic-button"
          >
            <Show when={recording()} fallback={<MicIcon />}>
              <StopIcon />
            </Show>
          </button>
        </div>
      }
    >
      <div class="flex items-center gap-2 flex-1 w-full" data-testid="voice-review">
        <div class="flex-1 flex items-center bg-canvas border border-edge-strong rounded-md focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/25">
          <input
            type="text"
            class="flex-1 bg-transparent text-ink pl-3 py-2 text-sm outline-none placeholder:text-ink-dim border-none"
            value={reviewText()}
            onInput={e => setReviewText(e.currentTarget.value)}
            onKeyDown={handleKeyDown}
            data-testid="voice-review-input"
          />
          <button
            class="bg-transparent border-none text-ink-dim text-sm cursor-pointer hover:text-ink hover:bg-muted rounded-md px-2 py-1 shrink-0 transition-all"
            onClick={handleCancel}
            aria-label="Cancel"
            data-testid="voice-review-cancel"
          >
            ✕
          </button>
        </div>
        <button
          class="bg-primary border-none text-on-primary w-10 h-10 rounded-full text-base cursor-pointer hover:bg-primary-hover active:scale-95 transition-all shrink-0"
          onClick={handleSend}
          aria-label="Send"
          data-testid="voice-review-send"
        >
          &#10148;
        </button>
      </div>
    </Show>
  );
};
