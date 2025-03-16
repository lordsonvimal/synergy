export interface ResponsiveValue {
  default?: string;
  sm?: string;
  md?: string;
  lg?: string;
  xl?: string;
}

// Helper function to format responsive CSS variables
export function formatCSSVar(
  value?: ResponsiveValue | string,
  defaultValue?: string
) {
  if (!value) return defaultValue || "auto";
  if (typeof value === "string") return value;
  return `var(--${value.default || "auto"})`;
}
