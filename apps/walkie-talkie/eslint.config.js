import js from "@eslint/js";
import tseslint from "typescript-eslint";

export default tseslint.config(
  js.configs.recommended,
  ...tseslint.configs.strict,
  {
    rules: {
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_" }
      ],
      "complexity": ["error", 5],
      "max-lines": ["error", { max: 500, skipBlankLines: true, skipComments: true }],
      "no-nested-ternary": "error"
    }
  },
  {
    ignores: ["dist/", "pwa/public/sw.js"]
  }
);
