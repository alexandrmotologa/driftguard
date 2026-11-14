import React, { useState } from 'react';
import { MarkovMatrix, MatrixPerturbation } from '../types';
import { GitCommit, Info } from 'lucide-react';

interface MarkovHeatmapProps {
  currentMatrix?: MarkovMatrix;
  baselineMatrix?: MarkovMatrix;
  perturbation?: MatrixPerturbation;
}

export const MarkovHeatmap: React.FC<MarkovHeatmapProps> = ({
  currentMatrix,
  baselineMatrix,
  perturbation,
}) => {
  const [mode, setMode] = useState<'current' | 'baseline' | 'diff'>('current');
  const [hoveredCell, setHoveredCell] = useState<{
    from: string;
    to: string;
    prob: number;
    baseProb?: number;
    count?: number;
  } | null>(null);

  // Collect all unique states
  const stateSet = new Set<string>();
  currentMatrix?.states.forEach((s) => stateSet.add(s));
  baselineMatrix?.states.forEach((s) => stateSet.add(s));

  const states = Array.from(stateSet).sort((a, b) => {
    if (a === '__START__') return -1;
    if (b === '__START__') return 1;
    if (a === '__END__') return 1;
    if (b === '__END__') return -1;
    return a.localeCompare(b);
  });

  const getCellData = (from: string, to: string) => {
    const currProb = currentMatrix?.probabilities?.[from]?.[to] ?? 0;
    const baseProb = baselineMatrix?.probabilities?.[from]?.[to] ?? 0;
    const count = currentMatrix?.raw_counts?.[from]?.[to] ?? 0;
    const diff = Math.abs(currProb - baseProb);

    return { currProb, baseProb, count, diff };
  };

  const getCellColor = (from: string, to: string) => {
    const { currProb, diff } = getCellData(from, to);

    if (mode === 'diff') {
      if (diff === 0) return 'bg-sre-850/40 text-slate-500';
      if (diff < 0.05) return 'bg-emerald-950/40 text-emerald-400 border border-emerald-900/40';
      if (diff < 0.15) return 'bg-amber-950/50 text-amber-300 border border-amber-800/60 font-semibold';
      return 'bg-rose-950/70 text-rose-300 border border-rose-700 font-bold';
    }

    const prob = mode === 'current' ? currProb : getCellData(from, to).baseProb;

    // Detect high self-loop
    if (from === to && from !== '__START__' && from !== '__END__' && prob > 0.20) {
      return 'bg-rose-950/60 text-rose-300 border border-rose-600/60 font-bold';
    }

    if (prob === 0) return 'bg-sre-850/30 text-slate-600';
    if (prob < 0.10) return 'bg-blue-950/20 text-blue-400';
    if (prob < 0.35) return 'bg-blue-900/40 text-blue-300 border border-blue-800/40';
    if (prob < 0.70) return 'bg-blue-800/60 text-blue-200 border border-blue-600/50 font-medium';
    return 'bg-blue-600 text-white font-bold';
  };

  if (!states.length) {
    return (
      <div className="bg-sre-900 border border-sre-800 rounded-xl p-6 text-center text-slate-400 text-sm">
        No Markov transition state data available for this workflow yet.
      </div>
    );
  }

  return (
    <div className="bg-sre-900 border border-sre-800 rounded-xl p-5">
      <div className="flex flex-wrap items-center justify-between gap-4 mb-4">
        <div>
          <div className="flex items-center gap-2">
            <GitCommit className="w-4 h-4 text-purple-400" />
            <h3 className="font-semibold text-sm text-white">
              Markov Tool Transition Matrix P(T<sub>j</sub> | T<sub>i</sub>)
            </h3>
          </div>
          <p className="text-xs text-slate-400 mt-0.5">
            Probability transitions between agent actions and guardrail checks
          </p>
        </div>

        {/* View Mode Toggle */}
        <div className="flex items-center bg-sre-850 border border-sre-700/60 rounded-lg p-0.5 text-xs font-mono">
          <button
            onClick={() => setMode('current')}
            className={`px-3 py-1 rounded-md transition cursor-pointer ${
              mode === 'current' ? 'bg-blue-600 text-white font-medium' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Current Window
          </button>
          <button
            onClick={() => setMode('baseline')}
            className={`px-3 py-1 rounded-md transition cursor-pointer ${
              mode === 'baseline' ? 'bg-blue-600 text-white font-medium' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Golden Baseline
          </button>
          <button
            onClick={() => setMode('diff')}
            className={`px-3 py-1 rounded-md transition cursor-pointer ${
              mode === 'diff' ? 'bg-purple-600 text-white font-medium' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            ΔM Divergence
          </button>
        </div>
      </div>

      {/* Matrix Table */}
      <div className="overflow-x-auto pb-2">
        <table className="w-full text-xs font-mono border-collapse">
          <thead>
            <tr>
              <th className="p-2 text-left text-slate-400 font-semibold bg-sre-850/60 border border-sre-800 rounded-tl-lg">
                From \ To
              </th>
              {states.map((to) => (
                <th
                  key={to}
                  className="p-2 text-center text-slate-300 font-semibold bg-sre-850/60 border border-sre-800 max-w-[110px] truncate"
                  title={to}
                >
                  {to.replace('__', '')}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {states.map((from) => (
              <tr key={from}>
                <td
                  className="p-2 text-left text-slate-300 font-medium bg-sre-850/40 border border-sre-800 max-w-[120px] truncate"
                  title={from}
                >
                  {from.replace('__', '')}
                </td>
                {states.map((to) => {
                  const data = getCellData(from, to);
                  const displayVal =
                    mode === 'diff'
                      ? data.diff.toFixed(2)
                      : mode === 'current'
                      ? data.currProb.toFixed(2)
                      : data.baseProb.toFixed(2);

                  return (
                    <td
                      key={to}
                      onMouseEnter={() =>
                        setHoveredCell({
                          from,
                          to,
                          prob: data.currProb,
                          baseProb: data.baseProb,
                          count: data.count,
                        })
                      }
                      onMouseLeave={() => setHoveredCell(null)}
                      className={`p-2 text-center transition cursor-crosshair border border-sre-800/80 ${getCellColor(
                        from,
                        to
                      )}`}
                    >
                      {displayVal === '0.00' ? '-' : displayVal}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Matrix Footnote / Hover Details */}
      <div className="mt-3 pt-3 border-t border-sre-800 flex items-center justify-between text-xs text-slate-400 font-mono">
        <div className="flex items-center gap-2">
          <Info className="w-3.5 h-3.5 text-slate-500" />
          {hoveredCell ? (
            <span>
              <span className="text-white font-semibold">{hoveredCell.from}</span> →{' '}
              <span className="text-white font-semibold">{hoveredCell.to}</span>: Current={hoveredCell.prob.toFixed(3)}
              {hoveredCell.baseProb !== undefined && ` | Baseline=${hoveredCell.baseProb.toFixed(3)}`}
              {hoveredCell.count !== undefined && ` (${hoveredCell.count} executions)`}
            </span>
          ) : (
            <span>Hover over any transition cell to inspect probability parameters</span>
          )}
        </div>
        {perturbation && (
          <span className="text-[11px] text-purple-400">
            Avg Perturbation: ΔM = {perturbation.mean_abs_deviation.toFixed(4)}
          </span>
        )}
      </div>
    </div>
  );
};
