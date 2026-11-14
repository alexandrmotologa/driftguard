import React from 'react';
import { ShieldCheck, AlertTriangle, AlertOctagon, TrendingUp, Cpu, GitCommit } from 'lucide-react';
import { EvaluationResult, Baseline } from '../types';

interface OverviewCardsProps {
  evaluation?: EvaluationResult;
  baseline?: Baseline;
  totalTraces: number;
}

export const OverviewCards: React.FC<OverviewCardsProps> = ({
  evaluation,
  baseline,
  totalTraces,
}) => {
  const status = evaluation?.status || 'NORMAL';
  const wasserstein = evaluation?.wasserstein_distance ?? 0;
  const maxWasserstein = baseline?.max_allowed_wasserstein_distance ?? 2.0;
  const stepMean = evaluation?.step_mean ?? 0;
  const stepStd = evaluation?.step_std ?? 0;
  const baselineMean = baseline?.expected_steps_mean ?? 0;
  const matrixDrift = evaluation?.matrix_deviation?.mean_abs_deviation ?? 0;
  const maxToolDrift = baseline?.max_tool_transition_drift ?? 0.18;
  const windowSize = evaluation?.window_size ?? 0;

  const wRatio = Math.min(100, Math.round((wasserstein / (maxWasserstein * 1.5)) * 100));

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
      {/* Status Card */}
      <div className="bg-sre-900 border border-sre-800 rounded-xl p-4 flex flex-col justify-between">
        <div className="flex items-center justify-between text-xs font-mono text-slate-400">
          <span>SENTINEL STATUS</span>
          {status === 'NORMAL' && <ShieldCheck className="w-4 h-4 text-emerald-400" />}
          {status === 'WARNING' && <AlertTriangle className="w-4 h-4 text-amber-400" />}
          {status === 'CRITICAL' && <AlertOctagon className="w-4 h-4 text-rose-500" />}
        </div>
        <div className="my-2">
          <div className="flex items-center gap-2">
            <span
              className={`inline-block w-2.5 h-2.5 rounded-full ${
                status === 'NORMAL'
                  ? 'bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]'
                  : status === 'WARNING'
                  ? 'bg-amber-400 shadow-[0_0_8px_rgba(251,191,36,0.8)]'
                  : 'bg-rose-500 shadow-[0_0_10px_rgba(244,63,94,0.9)] animate-pulse'
              }`}
            />
            <span
              className={`text-2xl font-bold tracking-tight ${
                status === 'NORMAL'
                  ? 'text-emerald-400'
                  : status === 'WARNING'
                  ? 'text-amber-400'
                  : 'text-rose-400'
              }`}
            >
              {status}
            </span>
          </div>
          <p className="text-[11px] text-slate-400 mt-1 truncate">
            {status === 'NORMAL' ? 'Trajectory within baseline bounds' : 'Behavioral drift detected'}
          </p>
        </div>
        <div className="text-[10px] text-slate-500 font-mono">
          Last evaluated: {evaluation ? new Date(evaluation.timestamp).toLocaleTimeString() : 'N/A'}
        </div>
      </div>

      {/* Wasserstein Distance Card */}
      <div className="bg-sre-900 border border-sre-800 rounded-xl p-4 flex flex-col justify-between">
        <div className="flex items-center justify-between text-xs font-mono text-slate-400">
          <span>WASSERSTEIN DIST</span>
          <TrendingUp className="w-4 h-4 text-blue-400" />
        </div>
        <div className="my-2">
          <div className="flex items-baseline gap-2">
            <span className="text-2xl font-bold text-white font-mono">
              {wasserstein.toFixed(3)}
            </span>
            <span className="text-xs text-slate-400 font-mono">
              / max {maxWasserstein.toFixed(1)}
            </span>
          </div>
          <div className="w-full bg-sre-800 h-1.5 rounded-full mt-2.5 overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-500 ${
                wasserstein > maxWasserstein
                  ? 'bg-rose-500'
                  : wasserstein > maxWasserstein * 0.75
                  ? 'bg-amber-400'
                  : 'bg-emerald-400'
              }`}
              style={{ width: `${wRatio}%` }}
            />
          </div>
        </div>
        <div className="text-[10px] text-slate-400 font-mono">
          {wasserstein > maxWasserstein ? 'Diverged from baseline' : 'Normal step density'}
        </div>
      </div>

      {/* Step Count Mean Card */}
      <div className="bg-sre-900 border border-sre-800 rounded-xl p-4 flex flex-col justify-between">
        <div className="flex items-center justify-between text-xs font-mono text-slate-400">
          <span>REASONING STEPS</span>
          <Cpu className="w-4 h-4 text-indigo-400" />
        </div>
        <div className="my-2">
          <div className="flex items-baseline gap-1.5">
            <span className="text-2xl font-bold text-white font-mono">
              {stepMean.toFixed(1)}
            </span>
            <span className="text-xs text-slate-400 font-mono">
              ± {stepStd.toFixed(1)}
            </span>
          </div>
          <div className="text-xs text-slate-400 mt-1">
            Baseline: <span className="text-slate-200 font-mono">{baselineMean.toFixed(1)}</span> steps
          </div>
        </div>
        <div className="text-[10px] text-slate-500 font-mono">
          Window sample: {windowSize} traces
        </div>
      </div>

      {/* Markov Perturbation Card */}
      <div className="bg-sre-900 border border-sre-800 rounded-xl p-4 flex flex-col justify-between">
        <div className="flex items-center justify-between text-xs font-mono text-slate-400">
          <span>MARKOV DRIFT (ΔM)</span>
          <GitCommit className="w-4 h-4 text-purple-400" />
        </div>
        <div className="my-2">
          <div className="flex items-baseline gap-2">
            <span className="text-2xl font-bold text-white font-mono">
              {matrixDrift.toFixed(4)}
            </span>
            <span className="text-xs text-slate-400 font-mono">
              / {maxToolDrift.toFixed(2)}
            </span>
          </div>
          <div className="text-xs text-slate-400 mt-1 truncate">
            {evaluation?.matrix_deviation?.novel_transitions?.length
              ? `${evaluation.matrix_deviation.novel_transitions.length} novel transitions`
              : 'Zero unobserved edges'}
          </div>
        </div>
        <div className="text-[10px] text-slate-500 font-mono truncate">
          Max edge: {evaluation?.matrix_deviation?.max_edge_from || 'none'}
        </div>
      </div>

      {/* Total Traces Card */}
      <div className="bg-sre-900 border border-sre-800 rounded-xl p-4 flex flex-col justify-between">
        <div className="flex items-center justify-between text-xs font-mono text-slate-400">
          <span>INGESTED RUNS</span>
          <span className="text-[10px] text-blue-400 uppercase font-mono">Active</span>
        </div>
        <div className="my-2">
          <span className="text-2xl font-bold text-white font-mono">
            {totalTraces.toLocaleString()}
          </span>
          <div className="text-xs text-slate-400 mt-1">
            Window size: <span className="text-slate-200 font-mono">{windowSize}</span> runs
          </div>
        </div>
        <div className="text-[10px] text-slate-500 font-mono">
          SQLite WAL persistence
        </div>
      </div>
    </div>
  );
};
