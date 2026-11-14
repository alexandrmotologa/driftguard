import React, { useState } from 'react';
import { Terminal, X, Play, CheckCircle2, AlertOctagon, Loader2, ArrowRight } from 'lucide-react';
import { TrajectoryDiff } from '../types';

interface SandboxModalProps {
  workflowId: string;
  isOpen: boolean;
  onClose: () => void;
}

export const SandboxModal: React.FC<SandboxModalProps> = ({
  workflowId,
  isOpen,
  onClose,
}) => {
  const [jsonInput, setJsonInput] = useState<string>(`{
  "workflow_id": "${workflowId}",
  "trace_id": "tr-sandbox-01",
  "steps": [
    {"tool_name": "lookup_order"},
    {"tool_name": "check_fraud_score"},
    {"tool_name": "execute_refund"},
    {"tool_name": "send_confirmation_email"}
  ]
}`);

  const [loading, setLoading] = useState<boolean>(false);
  const [result, setResult] = useState<{
    status: 'PASS' | 'FAIL';
    wasserstein_distance: number;
    explanation: string;
    diff: TrajectoryDiff;
  } | null>(null);

  if (!isOpen) return null;

  const loadSample = (type: 'clean' | 'loop' | 'bypass') => {
    if (type === 'clean') {
      setJsonInput(`{
  "workflow_id": "${workflowId}",
  "trace_id": "tr-sandbox-clean",
  "steps": [
    {"tool_name": "lookup_order"},
    {"tool_name": "check_fraud_score"},
    {"tool_name": "execute_refund"},
    {"tool_name": "send_confirmation_email"}
  ]
}`);
    } else if (type === 'loop') {
      setJsonInput(`{
  "workflow_id": "${workflowId}",
  "trace_id": "tr-sandbox-loop",
  "steps": [
    {"tool_name": "lookup_order"},
    {"tool_name": "lookup_order"},
    {"tool_name": "lookup_order"},
    {"tool_name": "check_fraud_score"},
    {"tool_name": "execute_refund"},
    {"tool_name": "send_confirmation_email"}
  ]
}`);
    } else {
      setJsonInput(`{
  "workflow_id": "${workflowId}",
  "trace_id": "tr-sandbox-bypass",
  "steps": [
    {"tool_name": "lookup_order"},
    {"tool_name": "execute_refund"},
    {"tool_name": "send_confirmation_email"}
  ]
}`);
    }
    setResult(null);
  };

  const handleTest = async () => {
    try {
      setLoading(true);
      const parsed = JSON.parse(jsonInput);
      const res = await fetch(`/api/v1/workflows/${workflowId}/test-trace`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(parsed),
      });

      if (!res.ok) {
        throw new Error('Test failed: ' + (await res.text()));
      }

      const data = await res.json();
      setResult({
        status: data.status,
        wasserstein_distance: data.evaluation.wasserstein_distance,
        explanation: data.evaluation.explanation,
        diff: data.trajectory_diff,
      });
    } catch (err: any) {
      alert('Error evaluating trace: ' + err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-sre-900 border border-sre-700 rounded-xl max-w-3xl w-full shadow-2xl overflow-hidden flex flex-col max-h-[88vh]">
        <div className="flex items-center justify-between p-4 border-b border-sre-800">
          <div className="flex items-center gap-2">
            <Terminal className="w-5 h-5 text-emerald-400" />
            <div>
              <h3 className="font-semibold text-white text-sm">
                Ad-Hoc Trace Testing Sandbox
              </h3>
              <p className="text-xs text-slate-400">
                Validate candidate agent traces against Golden Baseline without storing in DB
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

        <div className="p-5 overflow-y-auto space-y-4">
          {/* Quick template buttons */}
          <div className="flex items-center gap-2 text-xs font-mono">
            <span className="text-slate-400">Templates:</span>
            <button
              onClick={() => loadSample('clean')}
              className="px-2 py-1 rounded bg-sre-800 hover:bg-sre-700 text-slate-200 border border-sre-700 transition"
            >
              Clean Normal
            </button>
            <button
              onClick={() => loadSample('loop')}
              className="px-2 py-1 rounded bg-amber-950/40 hover:bg-amber-900/60 text-amber-300 border border-amber-800 transition"
            >
              Tool Loop
            </button>
            <button
              onClick={() => loadSample('bypass')}
              className="px-2 py-1 rounded bg-rose-950/40 hover:bg-rose-900/60 text-rose-300 border border-rose-800 transition"
            >
              Security Bypass
            </button>
          </div>

          <div>
            <textarea
              rows={8}
              value={jsonInput}
              onChange={(e) => setJsonInput(e.target.value)}
              className="w-full bg-sre-950 border border-sre-700 rounded-lg p-3 text-xs font-mono text-slate-200 focus:outline-none focus:border-blue-500"
              placeholder="Paste JSON trace payload here..."
            />
          </div>

          <div className="flex justify-end">
            <button
              onClick={handleTest}
              disabled={loading}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold cursor-pointer disabled:opacity-50"
            >
              {loading ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Play className="w-4 h-4 fill-white" />
              )}
              <span>Run Diagnostic Test</span>
            </button>
          </div>

          {/* Test Verdict & Diff */}
          {result && (
            <div className="pt-4 border-t border-sre-800 space-y-3 font-mono text-xs">
              <div
                className={`p-3 rounded-lg border flex items-center justify-between ${
                  result.status === 'PASS'
                    ? 'bg-emerald-950/30 border-emerald-800 text-emerald-300'
                    : 'bg-rose-950/40 border-rose-800 text-rose-300'
                }`}
              >
                <div className="flex items-center gap-2">
                  {result.status === 'PASS' ? (
                    <CheckCircle2 className="w-5 h-5 text-emerald-400" />
                  ) : (
                    <AlertOctagon className="w-5 h-5 text-rose-400" />
                  )}
                  <span className="font-bold text-sm">Verdict: {result.status}</span>
                </div>
                <div>W₁ Distance: {result.wasserstein_distance.toFixed(3)}</div>
              </div>

              <div className="text-slate-300 bg-sre-850 p-2.5 rounded border border-sre-800 text-[11px]">
                {result.explanation}
              </div>

              {/* Step comparison chips */}
              <div className="space-y-1">
                <span className="text-slate-400 font-sans text-xs font-semibold">
                  Trajectory Step Diff:
                </span>
                {result.diff.step_diffs.map((st, i) => (
                  <div
                    key={i}
                    className="p-2 rounded bg-sre-950 border border-sre-800/80 flex items-center justify-between text-[11px]"
                  >
                    <div className="flex items-center gap-2">
                      <span className="text-slate-500">#{st.step_index}</span>
                      <span className="font-bold text-white">
                        {st.observed_tool || st.expected_tool}
                      </span>
                    </div>
                    <span
                      className={`px-1.5 py-0.5 rounded text-[10px] font-bold ${
                        st.type === 'MATCH'
                          ? 'text-emerald-400 bg-emerald-950/60'
                          : 'text-rose-400 bg-rose-950/60'
                      }`}
                    >
                      {st.type}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
