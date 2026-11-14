import React from 'react';
import { HistogramBin } from '../types';
import { BarChart3 } from 'lucide-react';

interface StepHistogramProps {
  histogram?: HistogramBin[];
  wassersteinDistance: number;
}

export const StepHistogram: React.FC<StepHistogramProps> = ({
  histogram,
  wassersteinDistance,
}) => {
  if (!histogram || histogram.length === 0) {
    return (
      <div className="bg-sre-900 border border-sre-800 rounded-xl p-5 text-center text-slate-400 text-sm">
        No step distribution histogram data available.
      </div>
    );
  }

  // Find max probability for scaling
  let maxP = 0.05;
  histogram.forEach((bin) => {
    if (bin.baseline_p > maxP) maxP = bin.baseline_p;
    if (bin.current_p > maxP) maxP = bin.current_p;
  });

  return (
    <div className="bg-sre-900 border border-sre-800 rounded-xl p-5">
      <div className="flex items-center justify-between mb-4">
        <div>
          <div className="flex items-center gap-2">
            <BarChart3 className="w-4 h-4 text-blue-400" />
            <h3 className="font-semibold text-sm text-white">Step Count Distribution Shift</h3>
          </div>
          <p className="text-xs text-slate-400 mt-0.5">
            Golden Baseline vs Current Rolling Window Execution Length
          </p>
        </div>

        <div className="flex items-center gap-4 text-xs font-mono">
          <div className="flex items-center gap-1.5">
            <span className="w-3 h-3 rounded-sm bg-blue-500/80 inline-block" />
            <span className="text-slate-300">Baseline</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="w-3 h-3 rounded-sm bg-emerald-400/90 inline-block" />
            <span className="text-slate-300">Current Window</span>
          </div>
          <span className="text-slate-500">|</span>
          <span className="text-blue-400">W₁ = {wassersteinDistance.toFixed(3)}</span>
        </div>
      </div>

      {/* Bar Chart Visualization */}
      <div className="h-48 flex items-end gap-2 pt-6 pb-2 px-2 border-b border-sre-800">
        {histogram.map((bin) => {
          const baseHeight = Math.max(4, Math.round((bin.baseline_p / maxP) * 100));
          const currHeight = Math.max(4, Math.round((bin.current_p / maxP) * 100));

          const isElevated = bin.step >= 6 && bin.current_p > 0.08;

          return (
            <div key={bin.step} className="flex-1 flex flex-col items-center h-full justify-end group relative">
              {/* Tooltip */}
              <div className="absolute -top-12 opacity-0 group-hover:opacity-100 transition pointer-events-none bg-sre-800 border border-sre-700 text-[11px] font-mono px-2 py-1 rounded shadow-lg z-20 whitespace-nowrap">
                <div>Step {bin.step}</div>
                <div className="text-blue-400">Base: {(bin.baseline_p * 100).toFixed(1)}%</div>
                <div className="text-emerald-400">Curr: {(bin.current_p * 100).toFixed(1)}%</div>
              </div>

              {/* Dual Bars */}
              <div className="w-full flex items-end justify-center gap-1 h-full">
                {/* Baseline Bar */}
                <div
                  className="w-1/2 max-w-[14px] bg-blue-600/70 hover:bg-blue-500 rounded-t-sm transition-all duration-300"
                  style={{ height: `${baseHeight}%` }}
                />
                {/* Current Bar */}
                <div
                  className={`w-1/2 max-w-[14px] rounded-t-sm transition-all duration-300 ${
                    isElevated
                      ? 'bg-rose-500 hover:bg-rose-400'
                      : 'bg-emerald-400/90 hover:bg-emerald-300'
                  }`}
                  style={{ height: `${currHeight}%` }}
                />
              </div>

              {/* X-axis Label */}
              <span className="text-[11px] font-mono text-slate-400 mt-2">
                {bin.step}
              </span>
            </div>
          );
        })}
      </div>

      <div className="flex items-center justify-between text-[11px] text-slate-400 font-mono mt-2 px-1">
        <span>Reasoning turn count</span>
        <span>Relative frequency [0.0 - {maxP.toFixed(2)}]</span>
      </div>
    </div>
  );
};
