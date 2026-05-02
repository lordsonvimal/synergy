let audioContext: AudioContext | null = null;

function getContext(): AudioContext {
  if (!audioContext || audioContext.state === "closed") {
    audioContext = new AudioContext();
  }
  return audioContext;
}

export function playChime(): void {
  const ctx = getContext();
  if (ctx.state === "suspended") {
    ctx.resume();
  }

  const now = ctx.currentTime;
  const gain = ctx.createGain();
  gain.connect(ctx.destination);
  gain.gain.setValueAtTime(0.3, now);
  gain.gain.exponentialRampToValueAtTime(0.01, now + 0.4);

  const osc1 = ctx.createOscillator();
  osc1.type = "sine";
  osc1.frequency.setValueAtTime(880, now);
  osc1.connect(gain);
  osc1.start(now);
  osc1.stop(now + 0.2);

  const gain2 = ctx.createGain();
  gain2.connect(ctx.destination);
  gain2.gain.setValueAtTime(0, now);
  gain2.gain.setValueAtTime(0.3, now + 0.15);
  gain2.gain.exponentialRampToValueAtTime(0.01, now + 0.5);

  const osc2 = ctx.createOscillator();
  osc2.type = "sine";
  osc2.frequency.setValueAtTime(1175, now + 0.15);
  osc2.connect(gain2);
  osc2.start(now + 0.15);
  osc2.stop(now + 0.5);
}
