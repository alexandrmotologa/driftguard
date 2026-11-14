import React, { useState } from 'react';
import { Trace, TraceStep } from '../types';
import { Activity, ArrowRight, X, Clock, Layers, Sparkles } from 'lucide-react';

interface TraceExplorerProps {
  traces: Trace[];
}

export const TraceExplorer: React.FC<TraceExplorerProps> = ({ traces }) => {
  const [selectedTrace, setSelectedTrace] = useState<Trace | null>(null);

  return (
    <div className="bg-sre-900 border border-sre-800 rounded-xl p-5">
      <div className="flex items-center justify-between mb-4">
        <div>
          <div className="flex items-center gap-2">
            <Activity className="w-4 h-4 text-emerald-400" />
            <h3 className="font-semibold text-sm text-white">Recent Trajectory Runs</h3>
          </div>
          <p className="text-xs text-slate-400 mt-0.5">
            Streaming trace telemetry and reasoning turn sequences
          </p>
        </div>
        <span className="text-xs font-mono text-slate-400">
          Showing last {traces.length} executions
        </span>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs border-collapse">
          <thead>
            <tr className="border-b border-sre-800 text-slate-400 font-mono">
              <th className="py-2.5 px-3">Trace ID</th>
              <th className="py-2.5 px-3">Model</th>
              <th className="py-2.5 px-3">Steps</th>
              <th className="py-2.5 px-3">Trajectory Flow</th>
              <th className="py-2.5 px-3">Latency</th>
              <th className="py-2.5 px-3">Tokens</th>
              <th className="py-2.5 px-3">Timestamp</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-sre-800/60 font-mono">
            {traces.length === 0 ? (
              <tr>
                <td colSpan={7} className="py-8 text-center text-slate-500">
                  No trace executions recorded yet.
                </td>
              </tr>
            ) : (
              traces.map((tr) => (
                <tr
                  key={tr.trace_id}
                  onClick={() => setSelectedTrace(tr)}
                  className="hover:bg-sre-850/60 transition cursor-pointer group"
                >
                  <td className="py-2.5 px-3 text-blue-400 font-medium group-hover:underline">
                    {tr.trace_id.slice(0, 15)}...
                  </td>
                  <td className="py-2.5 px-3 text-slate-300">
                    {tr.model || 'unknown'}
                  </td>
                  <td className="py-2.5 px-3">
                    <span
                      className={`px-1.5 py-0.5 rounded text-[11px] font-bold ${
                        tr.step_count > 6
                          ? 'bg-rose-950/60 text-rose-300 border border-rose-800'
                          : 'bg-sre-800 text-slate-300'
                      }`}
                    >
                      {tr.step_count}
                    </span>
                  </td>
                  <td className="py-2.5 px-3 max-w-xs truncate">
                    <div className="flex items-center gap-1 overflow-x-hidden text-[11px]">
                      {tr.steps.map((s, idx) => (
                        <React.Fragment key={idx}>
                          <span
                            className="px-1.5 py-0.5 rounded bg-sre-800 text-slate-300 truncate max-w-[90px]"
                            title={s.tool_name}
                          >
                            {s.tool_name}
                          </span>
                          {idx < tr.steps.length - 1 && (
                            <ArrowRight className="w-2.5 h-2.5 text-slate-600 shrink-0" />
                          )}
                        </React.Fragment>
                      ))}
                    </div>
                  </td>
                  <td className="py-2.5 px-3 text-slate-300">
                    {tr.duration_ms}ms
                  </td>
                  <td className="py-2.5 px-3 text-slate-400">
                    {tr.token_usage.total_tokens || '-'}
                  </td>
                  <td className="py-2.5 px-3 text-slate-500">
                    {new Date(tr.timestamp).toLocaleTimeString()}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Trace Detail Modal */}
      {selectedTrace && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-sre-900 border border-sre-700 rounded-xl max-w-2xl w-full max-h-[85vh] flex flex-col shadow-2xl">
            <div className="flex items-center justify-between p-4 border-b border-sre-800">
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-mono text-sm font-bold text-white">
                    {selectedTrace.trace_id}
                  </span>
                  <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-400">
                    {selectedTrace.status}
                  </span>
                </div>
                <div className="text-xs text-slate-400 font-mono mt-0.5">
                  Workflow: {selectedTrace.workflow_id} | Model: {selectedTrace.model}
                </div>
              </div>
              <button
                onClick={() => setSelectedTrace(null)}
                className="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-sre-800 transition cursor-pointer"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="p-4 space-y-4 overflow-y-auto flex-1 font-mono text-xs">
              <div className="grid grid-cols-3 gap-2 bg-sre-850 p-3 rounded-lg border border-sre-800">
                <div>
                  <span className="text-slate-500">Duration:</span>{' '}
                  <span className="text-white font-semibold">{selectedTrace.duration_ms}ms</span>
                </div>
                <div>
                  <span className="text-slate-500">Steps:</span>{' '}
                  <span className="text-white font-semibold">{selectedTrace.step_count}</span>
                </div>
                <div>
                  <span className="text-slate-500">Tokens:</span>{' '}
                  <span className="text-white font-semibold">
                    {selectedTrace.token_usage.total_tokens}
                  </span>
                </div>
              </div>

              <div>
                <h4 className="font-sans font-semibold text-sm text-white mb-2">
                  Step Execution Sequence
                </h4>
                <div className="space-y-2">
                  {selectedTrace.steps.map((step: TraceStep) => (
                    <div
                      key={step.step_index}
                      className="p-3 rounded-lg bg-sre-850 border border-sre-800/80 space-y-1.5"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <span className="w-5 h-5 rounded-full bg-blue-600/30 text-blue-400 flex items-center justify-center font-bold text-[10px]">
                            {step.step_index}
                          </span>
                          <span className="font-bold text-white text-sm">
                            {step.tool_name}
                          </span>
                        </div>
                        <span className="text-slate-400 text-[11px]">
                          {step.duration_ms}ms | {step.tokens_used || 0} tokens
                        </span>
                      </div>

                      {step.tool_input && (
                        <div className="bg-sre-950 p-2 rounded text-slate-300 text-[11px] overflow-x-auto">
                          <span className="text-slate-500">Input:</span> {step.tool_input}
                        </div>
                      )}
                      {step.tool_output && (
                        <div className="bg-sre-950 p-2 rounded text-slate-300 text-[11px] overflow-x-auto">
                          <span className="text-slate-500">Output:</span> {step.tool_output}
                        </div>
                      )}
                      {step.error && (
                        <div className="bg-rose-950/40 border border-rose-800/50 p-2 rounded text-rose-300 text-[11px]">
                          <span className="font-bold text-rose-400">Error:</span> {step.error}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
