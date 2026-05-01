import { Component, createSignal, onCleanup, Show } from "solid-js";
import { createSTT } from "../lib/stt.js";
import { addToast } from "../lib/toast.js";

interface VoiceInputProps {
  onSend: (text: string) => void;
  onReviewChange?: (reviewing: boolean) => void;
}

export const VoiceInput: Component<VoiceInputProps> = (props) => {
  const [recording, setRecording] = createSignal(false);
  const [interim, setInterim] = createSignal("");
  const [accumulated, setAccumulated] = createSignal("");
  const [reviewText, setReviewText] = createSignal("");
  const [reviewing, setReviewing] = createSignal(false);

  const stt = createSTT({
    onInterim: (text) => {
      setInterim(text);
    },
    onFinal: (text) => {
      const updated = accumulated() + (accumulated() ? " " : "") + text;
      setAccumulated(updated);
      setInterim("");
    },
    onError: (err) => {
      if (err === "not-allowed") {
        addToast("Microphone permission denied", "error");
        setRecording(false);
      } else if (err === "network") {
        addToast("Network required for speech recognition", "error");
        setRecording(false);
      } else if (err === "no-speech") {
        // Ignore — auto-restart handles this
      } else if (err === "aborted") {
        // Ignore — user stopped
      } else {
        addToast(`Speech error: ${err}`, "error");
        setRecording(false);
      }
    },
    onEnd: () => {
      if (!recording() && !accumulated() && !interim()) {
        addToast("No speech detected", "warning");
      }
    }
  });

  onCleanup(() => {
    if (recording()) {
      stt?.stop();
    }
  });

  const handleToggle = (): void => {
    if (!stt) {
      addToast("Speech recognition not supported in this browser", "error");
      return;
    }
    if (recording()) {
      stt.stop();
      setRecording(false);
      const finalText = accumulated() + (interim() ? (accumulated() ? " " : "") + interim() : "");
      setInterim("");
      if (finalText.trim()) {
        setReviewText(finalText.trim());
        setReviewing(true);
        props.onReviewChange?.(true);
      }
      setAccumulated("");
    } else {
      setAccumulated("");
      setInterim("");
      setRecording(true);
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

  const displayText = (): string => {
    const acc = accumulated();
    const int = interim();
    if (acc && int) {
      return acc + " " + int;
    }
    return acc || int;
  };

  return (
    <Show
      when={reviewing()}
      fallback={
        <div class="flex items-center gap-2">
          <Show when={recording() && displayText()}>
            <span class="text-xs text-ink-secondary truncate max-w-[180px]">
              {displayText()}
            </span>
          </Show>
          <button
            class={`w-10 h-10 rounded-full border-none text-on-primary text-base cursor-pointer transition-transform active:scale-95 ${
              recording()
                ? "bg-error animate-pulse-recording"
                : "bg-primary hover:bg-primary-hover"
            }`}
            onClick={handleToggle}
            aria-label={recording() ? "Stop recording" : "Start recording"}
            data-testid="mic-button"
          >
            {recording() ? "⏹" : "🎙"}
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
