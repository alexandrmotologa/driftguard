import React, { useState } from 'react';
import { TimelinePoint } from '../types';
import { LineChart, Clock } from 'lucide-react';

interface DriftTimelineProps {
  timeline: TimelinePoint[];
  threshold: number;
}

export const DriftTimeline: React.FC<DriftTimelineProps> = ({
  timeline,
  threshold,
}) => {
  const [hoveredPoint, setHoveredPoint] = useState<TimelinePoint | null>(null);

  if (!timeline || timeline.length === 0) {
    return (
      <div className="bg-sre-900 border border-sre-800 rounded-xl p-5 text-center text-slate-400 text-xs font-mono">
        No historical drift evaluation points recorded yet.
      </div>
    );
  }

  // Calculate scales
  const maxW = Math.max(
    threshold * 1.6,
    ...timeline.map((p) => Math.max(p.wasserstein_distance, p.micro_wasserstein || 0))
  );

  const width = 800;
  const height = 180;
  const padding = { top: 20, right: 30, bottom: 30, left: 45 };

  const innerW = width - padding.left - padding.right;
  const innerH = height - padding.top - padding.bottom;

  const pointsCount = timeline.length;
  const getX = (idx: number) => {
    if (pointsCount <= 1) return padding.left + innerW / 2;
    return padding.left + (idx / (pointsCount - 1)) * innerW;
  };

  const getY = (val: number) => {
    const clamped = Math.max(0, Math.min(maxW, val));
    return padding.top + innerH - (clamped / maxW) * innerH;
  };

  // Generate SVG path for macro Wasserstein line
  let pathMacro = '';
  timeline.forEach((p, idx) => {
    const x = getX(idx);
    const y = getY(p.wasserstein_distance);
    pathMacro += idx === 0 ? `M ${x} ${y}` : ` L ${x} ${y}`;
  });

  // Generate SVG path for micro Wasserstein line
  let pathMicro = '';
  timeline.forEach((p, idx) => {
    const val = p.micro_wasserstein > 0 ? p.micro_wasserstein : p.wasserstein_distance;
    const x = getX(idx);
    const y = getY(val);
    pathMicro += idx === 0 ? `M ${x} ${y}` : ` L ${x} ${y}`;
  });

  const thresholdY = getY(threshold);

  return (
    <div className="bg-sre-900 border border-sre-800 rounded-xl p-5">
      <div className="flex items-center justify-between mb-3">
        <div>
          <div className="flex items-center gap-2">
            <LineChart className="w-4 h-4 text-blue-400" />
            <h3 className="font-semibold text-sm text-white">
              Historical Trajectory Drift Timeline
            </h3>
          </div>
          <p className="text-xs text-slate-400 mt-0.5">
            Sequential tracking of Wasserstein distance across recent audit evaluations
          </p>
        </div>

        <div className="flex items-center gap-4 text-xs font-mono">
          <div className="flex items-center gap-1.5">
            <span className="w-3 h-0.5 bg-blue-500 inline-block" />
            <span className="text-slate-300">Macro W₁</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="w-3 h-0.5 bg-cyan-400 inline-block" />
            <span className="text-slate-300">Micro W₁ (burst)</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="w-3 h-0.5 border-t border-dashed border-amber-400 inline-block" />
            <span className="text-amber-400">Threshold ({threshold.toFixed(2)})</span>
          </div>
        </div>
      </div>

      {/* SVG Chart */}
      <div className="relative w-full overflow-hidden">
        <svg
          viewBox={`0 0 ${width} ${height}`}
          className="w-full h-44 text-xs font-mono select-none"
        >
          {/* Y Axis Grid lines */}
          <line
            x1={padding.left}
            y1={padding.top}
            x2={width - padding.right}
            y2={padding.top}
            stroke="#1f2937"
            strokeDasharray="3 3"
          />
          <line
            x1={padding.left}
            y1={padding.top + innerH / 2}
            x2={width - padding.right}
            y2={padding.top + innerH / 2}
            stroke="#1f2937"
            strokeDasharray="3 3"
          />
          <line
            x1={padding.left}
            y1={padding.top + innerH}
            x2={width - padding.right}
            y2={padding.top + innerH}
            stroke="#374151"
          />

          {/* Threshold Line */}
          <line
            x1={padding.left}
            y1={thresholdY}
            x2={width - padding.right}
            y2={thresholdY}
            stroke="#f59e0b"
            strokeWidth="1.5"
            strokeDasharray="4 4"
          />

          {/* Micro line (Cyan) */}
          <path
            d={pathMicro}
            fill="none"
            stroke="#22d3ee"
            strokeWidth="1.5"
            opacity="0.8"
          />

          {/* Macro line (Blue) */}
          <path
            d={pathMacro}
            fill="none"
            stroke="#3b82f6"
            strokeWidth="2.5"
          />

          {/* Interactive Data Points */}
          {timeline.map((p, idx) => {
            const x = getX(idx);
            const y = getY(p.wasserstein_distance);
            const isCritical = p.status === 'CRITICAL';
            const isWarning = p.status === 'WARNING';

            return (
              <circle
                key={idx}
                cx={x}
                cy={y}
                r={hoveredPoint === p ? 5 : 3.5}
                fill={isCritical ? '#ef4444' : isWarning ? '#f59e0b' : '#3b82f6'}
                stroke="#111827"
                strokeWidth="1.5"
                className="cursor-pointer transition-all"
                onMouseEnter={() => setHoveredPoint(p)}
                onMouseLeave={() => setHoveredPoint(null)}
              />
            );
          })}

          {/* Y Axis Labels */}
          <text x={padding.left - 8} y={padding.top + 4} textAnchor="end" fill="#9ca3af">
            {maxW.toFixed(1)}
          </text>
          <text x={padding.left - 8} y={thresholdY + 4} textAnchor="end" fill="#f59e0b">
            {threshold.toFixed(1)}
          </text>
          <text x={padding.left - 8} y={padding.top + innerH} textAnchor="end" fill="#9ca3af">
            0.0
          </text>
        </svg>

        {/* Floating Tooltip */}
        {hoveredPoint && (
          <div className="absolute top-2 right-4 bg-sre-800 border border-sre-700 p-2.5 rounded-lg shadow-xl text-xs font-mono space-y-1 z-10">
            <div className="flex items-center gap-1.5 text-slate-400">
              <Clock className="w-3 h-3" />
              <span>{new Date(hoveredPoint.timestamp).toLocaleTimeString()}</span>
            </div>
            <div className="text-white">
              Macro W₁: <span className="font-bold text-blue-400">{hoveredPoint.wasserstein_distance.toFixed(3)}</span>
            </div>
            {hoveredPoint.micro_wasserstein > 0 && (
              <div className="text-white">
                Micro W₁: <span className="font-bold text-cyan-400">{hoveredPoint.micro_wasserstein.toFixed(3)}</span>
              </div>
            )}
            <div className="text-white">
              Markov ΔM: <span className="font-bold text-purple-400">{hoveredPoint.matrix_drift.toFixed(4)}</span>
            </div>
            <div>
              Status:{' '}
              <span
                className={`font-bold ${
                  hoveredPoint.status === 'CRITICAL'
                    ? 'text-rose-400'
                    : hoveredPoint.status === 'WARNING'
                    ? 'text-amber-400'
                    : 'text-emerald-400'
                }`}
              >
                {hoveredPoint.status}
              </span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
