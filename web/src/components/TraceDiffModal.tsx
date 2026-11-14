import React from 'react';
import { TrajectoryDiff, DiffStep } from '../types';
import { GitCompare, X, ShieldAlert, CheckCircle2, AlertTriangle, ArrowRight } from 'lucide-react';

interface TraceDiffModalProps {
  diff: TrajectoryDiff | null;
  isOpen: boolean;
  onClose: () => void;
}

export const TraceDiffModal: React.FC<TraceDiffModalProps> = ({
  diff,
  isOpen,
  onClose,
}) => {
  if (!isOpen || !diff) return null;

  return (
    <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-sre-900 border border-sre-700 rounded-xl max-w-2xl w-full shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
        <div className="flex items-center justify-between p-4 border-b border-sre-800">
          <div className="flex items-center gap-2">
            <GitCompare className="w-5 h-5 text-purple-400" />
            <div>
              <h3 className="font-semibold text-white text-sm">
                Visual Trajectory Differential Inspector
              </h3>
              <p className="text-xs text-slate-400 font-mono">
                Trace: {diff.trace_id}
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-sre-800 transition cursor-pointer"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-5 overflow-y-auto space-y-4 text-xs font-mono">
          {/* Guardrail Bypass Warning Banner */}
          {diff.has_guardrail_bypass && (
            <div className="p-3 rounded-lg bg-rose-950/40 border border-rose-800 flex items-start gap-2.5 text-rose-200">
              <ShieldAlert className="w-4 h-4 text-rose-400 shrink-0 mt-0.5" />
              <div>
                <span className="font-bold">CRITICAL POLICY BREACH:</span> This execution trajectory bypassed mandatory verification steps before executing privileged actions.
              </div>
            </div>
          )}

          {/* Metric Summary Chips */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-2 bg-sre-850 p-3 rounded-lg border border-sre-800 text-slate-300">
            <div>
              <span className="text-slate-500">Total Deviations:</span>{' '}
              <span className={`font-bold ${diff.total_deviations > 0 ? 'text-amber-400' : 'text-emerald-400'}`}>
                {diff.total_deviations}
              </span>
            </div>
            <div>
              <span className="text-slate-500">Loops Detected:</span>{' '}
              <span className={`font-bold ${diff.loop_count > 0 ? 'text-rose-400' : 'text-slate-200'}`}>
                {diff.loop_count}
              </span>
            </div>
            <div>
              <span className="text-slate-500">Omitted Steps:</span>{' '}
              <span className={`font-bold ${diff.omission_count > 0 ? 'text-rose-400' : 'text-slate-200'}`}>
                {diff.omission_count}
              </span>
            </div>
          </div>

          {/* Step Diff Table */}
          <div className="space-y-2">
            <h4 className="font-sans font-semibold text-xs text-slate-400 uppercase tracking-wider">
              Step Comparison (Expected vs Observed)
            </h4>

            <div className="space-y-1.5">
              {diff.step_diffs.map((step: DiffStep, idx: number) => {
                let badgeColor = 'bg-slate-800 text-slate-300 border-slate-700';
                if (step.type === 'MATCH') badgeColor = 'bg-emerald-950/60 text-emerald-300 border-emerald-800';
                if (step.type === 'LOOP') badgeColor = 'bg-rose-950/70 text-rose-300 border-rose-700';
                if (step.type === 'OMISSION') badgeColor = 'bg-amber-950/70 text-amber-300 border-amber-700';
                if (step.type === 'INSERTION') badgeColor = 'bg-purple-950/70 text-purple-300 border-purple-700';
                if (step.type === 'REPLACEMENT') badgeColor = 'bg-orange-950/70 text-orange-300 border-orange-700';

                return (
                  <div
                    key={idx}
                    className="p-3 rounded-lg bg-sre-850/80 border border-sre-800 flex items-start justify-between gap-3"
                  >
                    <div className="space-y-1 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="text-slate-500">#{step.step_index}</span>
                        {step.expected_tool && (
                          <span className="px-1.5 py-0.5 rounded bg-sre-900 border border-sre-700 text-slate-300">
                            {step.expected_tool}
                          </span>
                        )}
                        <ArrowRight className="w-3 h-3 text-slate-500" />
                        <span className="px-1.5 py-0.5 rounded bg-sre-950 border border-sre-700 font-bold text-white">
                          {step.observed_tool || '<omitted>'}
                        </span>
                      </div>
                      <div className="text-[11px] text-slate-400">
                        {step.explanation}
                      </div>
                    </div>

                    <span
                      className={`text-[10px] font-bold px-2 py-0.5 rounded border ${badgeColor}`}
                    >
                      {step.type}
                    </span>
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        <div className="p-4 border-t border-sre-800 flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-1.5 rounded-lg bg-sre-800 hover:bg-sre-700 text-slate-200 text-xs font-medium cursor-pointer"
          >
            Close Inspector
          </button>
        </div>
      </div>
    </div>
  );
};
