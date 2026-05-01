import { execSync } from "child_process";

export interface BatteryInfo {
  level: number;
  charging: boolean;
}

export function getBattery(): BatteryInfo {
  try {
    const output = execSync(
      "pmset -g batt",
      { encoding: "utf-8", timeout: 3000 }
    );
    const match = output.match(/(\d+)%/);
    const level = match ? parseInt(match[1]!, 10) : 0;
    const charging = /charging/.test(output) && !/discharging/.test(output);
    return { level, charging };
  } catch {
    return { level: 0, charging: false };
  }
}
