import "./style.css";

// Theme Logic
if (
  localStorage.getItem("color-theme") === "dark" ||
  (!("color-theme" in localStorage) &&
    window.matchMedia("(prefers-color-scheme: dark)").matches)
) {
  document.documentElement.classList.add("dark");
} else {
  document.documentElement.classList.remove("dark");
}

document.addEventListener("DOMContentLoaded", function () {
  // Hamburger Logic
  const btn = document.getElementById("hamburger-btn");
  const menu = document.getElementById("mobile-menu");
  const icon = document.getElementById("hamburger-icon");
  const links = document.querySelectorAll(".mobile-link");

  btn.addEventListener("click", e => {
    e.stopPropagation();
    const isHidden = menu.classList.contains("hidden");
    menu.classList.toggle("hidden");
    icon.textContent = isHidden ? "close" : "menu";
    icon.style.color = isHidden ? "#ef4444" : "";
  });

  // Dark Mode Toggle Logic
  const themeToggleBtns = document.getElementsByName("theme-toggle");

  themeToggleBtns.forEach(btn => {
    btn.addEventListener("click", function () {
      // 1. Toggle the 'dark' class on <html>
      if (document.documentElement.classList.contains("dark")) {
        document.documentElement.classList.remove("dark");
        localStorage.setItem("color-theme", "light");
      } else {
        document.documentElement.classList.add("dark");
        localStorage.setItem("color-theme", "dark");
      }
    });
    // Note: The icons will now flip automatically because of
    // the 'dark:hidden' and 'dark:block' classes in the HTML!
  });
});
