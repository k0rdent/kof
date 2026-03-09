import { Theme } from "@/components/shared/ThemeToggle";
import { create } from "zustand";

export const useTheme = create<ThemeProviderState>()((set) => ({
  theme: (localStorage.getItem("vite-ui-theme") as Theme) || "system",
  setTheme: (theme: Theme) => {
    localStorage.setItem("vite-ui-theme", theme);
    set({ theme });
  },
}));

type ThemeProviderState = {
  theme: Theme;
  setTheme: (theme: Theme) => void;
};
