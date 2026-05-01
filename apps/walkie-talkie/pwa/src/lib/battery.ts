interface BatteryManager {
  level: number;
  charging: boolean;
  addEventListener: (event: string, handler: () => void) => void;
  removeEventListener: (event: string, handler: () => void) => void;
}

interface NavigatorWithBattery extends Navigator {
  getBattery?: () => Promise<BatteryManager>;
}

export interface BatteryStatus {
  level: number;
  charging: boolean;
  supported: boolean;
}

export async function getBatteryStatus(): Promise<BatteryStatus> {
  const nav = navigator as NavigatorWithBattery;

  if (!nav.getBattery) {
    return { level: 0, charging: false, supported: false };
  }

  const battery = await nav.getBattery();
  return {
    level: Math.round(battery.level * 100),
    charging: battery.charging,
    supported: true
  };
}
