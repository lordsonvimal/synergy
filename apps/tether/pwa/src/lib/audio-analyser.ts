export interface AudioAnalyser {
  start: () => Promise<void>;
  stop: () => void;
  getLevel: () => number;
}

export function createAudioAnalyser(): AudioAnalyser {
  let context: AudioContext | null = null;
  let analyser: AnalyserNode | null = null;
  let source: MediaStreamAudioSourceNode | null = null;
  let stream: MediaStream | null = null;
  let dataArray: Uint8Array<ArrayBuffer> | null = null;

  const start = async (): Promise<void> => {
    stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    context = new AudioContext();
    analyser = context.createAnalyser();
    analyser.fftSize = 256;
    analyser.smoothingTimeConstant = 0.7;
    source = context.createMediaStreamSource(stream);
    source.connect(analyser);
    dataArray = new Uint8Array(analyser.frequencyBinCount) as Uint8Array<ArrayBuffer>;
  };

  const stop = (): void => {
    source?.disconnect();
    source = null;
    stream?.getTracks().forEach(t => t.stop());
    stream = null;
    if (context?.state !== "closed") {
      context?.close();
    }
    context = null;
    analyser = null;
    dataArray = null;
  };

  const getLevel = (): number => {
    if (!analyser || !dataArray) return 0;
    analyser.getByteFrequencyData(dataArray);
    let sum = 0;
    for (let i = 0; i < dataArray.length; i++) {
      sum += dataArray[i]!;
    }
    return sum / (dataArray.length * 255);
  };

  return { start, stop, getLevel };
}
