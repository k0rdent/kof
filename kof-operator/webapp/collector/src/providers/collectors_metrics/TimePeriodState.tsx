import { create } from "zustand";

export const TIME_PERIOD: TimePeriod[] = [
  { value: 60, text: "1m" },
  { value: 300, text: "5m" },
  { value: 600, text: "10m" },
  { value: 900, text: "15m" },
  { value: 1800, text: "30m" },
  { value: 2700, text: "45m" },
  { value: 3600, text: "60m" },
];

export interface TimePeriod {
  value: number;
  text: string;
}

type TimePeriodState = {
  timePeriod: TimePeriod;
  setTimePeriod: (period: TimePeriod) => void;
};

export const useTimePeriod = create<TimePeriodState>()((set) => {
  const setTimePeriod = (period: TimePeriod) => {
    set({ timePeriod: period });
  };

  return {
    timePeriod: TIME_PERIOD[0],
    setTimePeriod,
  };
});
