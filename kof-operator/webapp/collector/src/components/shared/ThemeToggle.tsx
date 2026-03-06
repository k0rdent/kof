import { useTheme } from "@/providers/ThemeProvider";
import { JSX, useEffect } from "react";
import { Button } from "../generated/ui/button";
import { Moon, Sun } from "lucide-react";

export type Theme = "dark" | "light" | "system";

const ThemeToggle = (): JSX.Element => {
  const { theme, setTheme } = useTheme();

  useEffect(() => {
    const root = window.document.documentElement;

    root.classList.remove("light", "dark");

    if (theme === "system") {
      const systemTheme = window.matchMedia("(prefers-color-scheme: dark)").matches
        ? "dark"
        : "light";

      root.classList.add(systemTheme);
      return;
    }

    root.classList.add(theme);
  }, [theme]);

  return (
    <Button
      onClick={() => setTheme(theme === "light" ? "dark" : "light")}
      variant="outline"
      size="icon"
      className="cursor-pointer"
    >
      {theme === "light" ? (
        <Moon className="h-[1.2rem] w-[1.2rem] scale-100 rotate-0 transition-all dark:scale-0 dark:-rotate-90" />
      ) : (
        <Sun className="absolute h-[1.2rem] w-[1.2rem] scale-0 rotate-90 transition-all dark:scale-100 dark:rotate-0" />
      )}
    </Button>
  );
};

export default ThemeToggle;
