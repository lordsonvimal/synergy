import { Component, createSignal, onCleanup, onMount, For } from "solid-js";
import { createAudioAnalyser, AudioAnalyser } from "../lib/audio-analyser.js";

const BAR_COUNT = 5;
const MIN_HEIGHT = 4;
const MAX_HEIGHT = 24;

export const Waveform: Component = () => {
  const [bars, setBars] = createSignal<number[]>(
    Array(BAR_COUNT).fill(MIN_HEIGHT)
  );
  let analyser: AudioAnalyser | null = null;
  let frameId: number | undefined;

  const animate = (): void => {
    if (!analyser) return;
    const level = analyser.getLevel();
    const newBars = Array.from({ length: BAR_COUNT }, (_, i) => {
      const offset = Math.sin(Date.now() / 150 + i * 1.2) * 0.3 + 0.7;
      const height = MIN_HEIGHT + (MAX_HEIGHT - MIN_HEIGHT) * level * offset;
      return Math.max(MIN_HEIGHT, Math.min(MAX_HEIGHT, height));
    });
    setBars(newBars);
    frameId = requestAnimationFrame(animate);
  };

  onMount(async () => {
    analyser = createAudioAnalyser();
    try {
      await analyser.start();
      frameId = requestAnimationFrame(animate);
    } catch {
      setBars(Array(BAR_COUNT).fill(MIN_HEIGHT));
    }
  });

  onCleanup(() => {
    if (frameId !== undefined) {
      cancelAnimationFrame(frameId);
    }
    analyser?.stop();
    analyser = null;
  });

  return (
    <div
      class="flex items-center gap-0.5 h-6"
      aria-hidden="true"
      data-testid="waveform"
    >
      <For each={bars()}>
        {height => (
          <span
            class="w-1 rounded-full bg-error transition-all duration-100"
            style={{ height: `${height}px` }}
          />
        )}
      </For>
    </div>
  );
};
