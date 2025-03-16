import { JSX, ParentComponent } from "solid-js";
import styles from "./layout.module.scss";
import { formatCSSVar, ResponsiveValue } from "./layout_utils";

interface SectionProps {
  gap?: ResponsiveValue | string; // Space between columns
  gutter?: ResponsiveValue | string; // Padding around the row
  align?: "start" | "center" | "end";
  overflow?: "visible" | "hidden" | "scroll" | "auto";
  height?: ResponsiveValue | string;
  maxHeight?: ResponsiveValue | string;
  minHeight?: ResponsiveValue | string;
  children: JSX.Element;
}

export const Section: ParentComponent<SectionProps> = props => {
  return (
    <section
      class={styles.layoutRow}
      style={{
        "--row-gap": formatCSSVar(props.gap, "8px"),
        "--row-gutter": formatCSSVar(props.gutter, "16px"),
        "--row-height": formatCSSVar(props.height),
        "--row-max-height": formatCSSVar(props.maxHeight),
        "--row-min-height": formatCSSVar(props.minHeight),
        "align-items": props.align || "start",
        height: "var(--row-height)",
        "max-height": "var(--row-max-height)",
        "min-height": "var(--row-min-height)",
        overflow: props.overflow || "visible",
        padding: "var(--row-gutter)"
      }}
    >
      {props.children}
    </section>
  );
};
