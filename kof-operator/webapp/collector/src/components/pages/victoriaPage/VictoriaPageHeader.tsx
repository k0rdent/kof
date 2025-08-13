import { JSX } from "react";
import SelectItems from "../collectorPage/components/collector-details/Select";
import { TIME_PERIOD, TimePeriod, useTimePeriod } from "@/providers/collectors_metrics/TimePeriodState";

const VictoriaPageHeader = (): JSX.Element => {
  const { timePeriod, setTimePeriod } = useTimePeriod();

  const onTimePeriodSelect = (timePeriod: string): void => {
    const newTimePeriod: TimePeriod | undefined = TIME_PERIOD.find(
      (period) => period.text == timePeriod
    );
    if (newTimePeriod) {
      setTimePeriod(newTimePeriod);
    }
  };

    return (
    <header className="flex justify-between">
      <h1 className="font-bold text-3xl">VictoriaMetrics/Logs Metrics</h1>
      <div className="flex gap-2 items-center">
        <span>Trend Period:</span>
        <SelectItems
          items={TIME_PERIOD.map((t) => t.text)}
          callbackFn={onTimePeriodSelect}
          disabled={false}
          value={timePeriod.text}
          fieldStyle="w-[80px]"
        ></SelectItems>
      </div>
    </header>
    )
}

export default VictoriaPageHeader