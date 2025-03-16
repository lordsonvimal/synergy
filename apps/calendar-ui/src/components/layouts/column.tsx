import { JSX, ParentComponent } from "solid-js";
import { Dynamic } from "solid-js/web";
import { formatCSSVar, ResponsiveValue } from "./layout_utils";
import styles from "./layout.module.scss";

interface ColumnProps {
  as?: keyof JSX.IntrinsicElements;
  className?: string;
  direction?: "row" | "column";
  span?: 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;
  start?: number;
  end?: number;
  overflow?: "visible" | "hidden" | "scroll" | "auto";
  height?: ResponsiveValue | string;
  maxHeight?: ResponsiveValue | string;
  minHeight?: ResponsiveValue | string;
  textAlign?: "left" | "center" | "right" | "justify";
  alignItems?: "start" | "center" | "end" | "stretch";
  justifyContent?:
    | "start"
    | "center"
    | "end"
    | "space-between"
    | "space-around"
    | "space-evenly";
  padding?: ResponsiveValue | string;
  margin?: ResponsiveValue | string;
  flexGrow?: number;
  flexShrink?: number;
  flexBasis?: string;
  gap?: ResponsiveValue | string;
  children: JSX.Element;
}

export const Column: ParentComponent<ColumnProps> = props => {
  const span = props.span ?? 12;
  let gridColumn = `span ${span}`;

  if (props.start && props.end) {
    gridColumn = `${props.start} / ${props.end}`;
  } else if (props.start) {
    gridColumn = `${props.start} / span ${span}`;
  } else if (props.end) {
    gridColumn = `span ${span} / ${props.end}`;
  }

  return (
    <Dynamic
      component={props.as || "div"}
      class={`${styles.column} ${props.className || ""}`}
      style={{
        "--column-height": formatCSSVar(props.height),
        "--column-max-height": formatCSSVar(props.maxHeight),
        "--column-min-height": formatCSSVar(props.minHeight),
        "--column-padding": formatCSSVar(props.padding),
        "--column-margin": formatCSSVar(props.margin),
        "--column-gap": formatCSSVar(props.gap, "0px"),
        "flex-direction": props.direction || "column",
        "grid-column": gridColumn,
        overflow: props.overflow || "visible",
        "text-align": props.textAlign,
        "align-items": props.alignItems,
        "justify-content": props.justifyContent,
        "flex-grow": props.flexGrow,
        "flex-shrink": props.flexShrink,
        "flex-basis": props.flexBasis,
        gap: "var(--column-gap)" // 🆕 Apply gap
      }}
    >
      {props.children}
    </Dynamic>
  );
};
