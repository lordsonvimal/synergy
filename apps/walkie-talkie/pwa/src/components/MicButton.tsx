import { Component, createSignal, onCleanup, Show } from "solid-js";
import { createSTT } from "../lib/stt.js";

interface MicButtonProps {
  onSend: (text: string) => void;
}

export const MicButton: Component<MicButtonProps> = (props) => {
  const [recording, setRecording] = createSignal(false);
  const [interim, setInterim] = createSignal("");

  const stt = createSTT({
    onInterim: (text) => setInterim(text),
    onFinal: (text) => {
      props.onSend(text);
      setInterim("");
      setRecording(false);
    },
    onError: () => {
      setRecording(false);
      setInterim("");
    }
  });

  onCleanup(() => {
    if (recording()) {
      stt?.stop();
    }
  });

  const handlePointerDown = (): void => {
    if (!stt) {
      return;
    }
    setRecording(true);
    stt.start();
  };

  const handlePointerUp = (): void => {
    if (!stt || !recording()) {
      return;
    }
    stt.stop();
  };

  return (
    <div class="ml-auto flex flex-col items-center gap-1">
      <Show when={recording() && interim()}>
        <div class="text-[13px] text-text-secondary max-w-[200px] text-center">
          {interim()}
        </div>
      </Show>
      <button
        class={`w-16 h-16 rounded-full border-none text-white text-2xl cursor-pointer touch-none transition-transform active:scale-95 ${
          recording()
            ? "bg-recording animate-pulse-recording"
            : "bg-accent"
        }`}
        onPointerDown={handlePointerDown}
        onPointerUp={handlePointerUp}
        onPointerLeave={handlePointerUp}
      >
        &#127908;
      </button>
      <Show when={recording()}>
        <span class="text-xs text-recording font-semibold">Recording...</span>
      </Show>
    </div>
  );
};
