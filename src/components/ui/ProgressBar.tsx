interface ProgressBarProps {
  value: number;
  max: number;
  color?: string;
  /** When true, high values are good (e.g. cache hit ratio). Disables red/orange overrides. */
  highIsGood?: boolean;
}

export function ProgressBar({ value, max, color = "indigo", highIsGood = false }: ProgressBarProps) {
  const pct = max > 0 ? Math.min((value / max) * 100, 100) : 0;
  let barColor: string;
  if (highIsGood) {
    // High = good: low values are concerning, high values stay the requested color
    barColor = pct < 50 ? "bg-red-500" : pct < 80 ? "bg-orange-500" : `bg-${color}-500`;
  } else {
    // High = bad (default): high utilization turns red
    barColor = pct > 90 ? "bg-red-500" : pct > 70 ? "bg-orange-500" : `bg-${color}-500`;
  }
  return (
    <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2.5">
      <div
        className={`${barColor} h-2.5 rounded-full transition-all duration-300`}
        style={{ width: `${pct}%` }}
      />
    </div>
  );
}
