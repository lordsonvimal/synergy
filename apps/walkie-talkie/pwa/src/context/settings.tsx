import {
  Component,
  JSX,
  createContext,
  createSignal,
  useContext
} from "solid-js";

export interface Settings {
  host: string;
  port: number;
  autoRead: boolean;
  ttsSpeed: number;
  ttsVoice: string;
  theme: "dark" | "light";
  fontSize: "small" | "medium" | "large";
}

const DEFAULT_SETTINGS: Settings = {
  host: "192.168.1.1",
  port: 5100,
  autoRead: true,
  ttsSpeed: 1.0,
  ttsVoice: "",
  theme: "dark",
  fontSize: "medium"
};

const STORAGE_KEY = "walkie-talkie-settings";

function loadSettings(): Settings {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (!stored) {
    return DEFAULT_SETTINGS;
  }
  return { ...DEFAULT_SETTINGS, ...JSON.parse(stored) };
}

function saveSettings(settings: Settings): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
}

interface SettingsContextValue {
  settings: () => Settings;
  updateSettings: (partial: Partial<Settings>) => void;
}

const SettingsContext = createContext<SettingsContextValue>();

export const SettingsProvider: Component<{ children: JSX.Element }> = (
  props
) => {
  const [settings, setSettings] = createSignal<Settings>(loadSettings());

  const updateSettings = (partial: Partial<Settings>): void => {
    const updated = { ...settings(), ...partial };
    setSettings(updated);
    saveSettings(updated);
  };

  return (
    <SettingsContext.Provider value={{ settings, updateSettings }}>
      {props.children}
    </SettingsContext.Provider>
  );
};

export function useSettings(): SettingsContextValue {
  const ctx = useContext(SettingsContext);
  if (!ctx) {
    throw new Error("useSettings must be used within SettingsProvider");
  }
  return ctx;
}
