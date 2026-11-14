import React, { useState } from 'react';
import { Zap, X, ShieldCheck, Repeat, ShieldAlert, Sparkles, Loader2 } from 'lucide-react';

interface SimulateModalProps {
  workflowId: string;
  isOpen: boolean;
  onClose: () => void;
  onSimulate: (scenario: string, count: number) => Promise<void>;
}

export const SimulateModal: React.FC<SimulateModalProps> = ({
  workflowId,
  isOpen,
  onClose,
  onSimulate,
}) => {
  const [scenario, setScenario] = useState<string>('loop');
  const [count, setCount] = useState<number>(25);
  const [loading, setLoading] = useState<boolean>(false);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await onSimulate(scenario, count);
      onClose();
    } catch (err) {
      console.error('Simulation failed:', err);
    } finally {
      setLoading(false);
    }
  };

  const scenarios = [
    {
      id: 'clean',
      title: 'Clean Golden Production Traffic',
      icon: ShieldCheck,
      color: 'text-emerald-400',
      description: 'Generates normal healthy trajectories (3-5 steps) strictly adhering to baseline.',
      defaultCount: 30,
    },
    {
      id: 'loop',
      title: 'Reasoning Loop Degradation',
      icon: Repeat,
      color: 'text-amber-400',
      description: 'Simulates checkpoint update causing cyclical tool loops and step explosion (8-15 steps).',
      defaultCount: 25,
    },
    {
      id: 'bypass',
      title: 'Guardrail Policy Bypass',
      icon: ShieldAlert,
      color: 'text-rose-400',
      description: 'Directly violates forbidden_transitions by skipping fraud verification before refunding.',
      defaultCount: 15,
    },
    {
      id: 'hallucinate',
      title: 'Hallucinated Tool Anomaly',
      icon: Sparkles,
      color: 'text-purple-400',
      description: 'Agent invokes unknown non-existent tools (e.g. auto_resolve_v2) failing downstream logic.',
      defaultCount: 10,
    },
  ];

  return (
    <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-sre-900 border border-sre-700 rounded-xl max-w-lg w-full shadow-2xl overflow-hidden">
        <div className="flex items-center justify-between p-4 border-b border-sre-800">
          <div className="flex items-center gap-2">
            <Zap className="w-5 h-5 text-blue-400" />
            <h3 className="font-semibold text-white">Simulate Agent Trajectory Drift</h3>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-sre-800 transition cursor-pointer"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          <div className="text-xs text-slate-300">
            Target Workflow: <span className="font-mono font-bold text-white">{workflowId}</span>
          </div>

          <div className="space-y-2">
            <label className="text-xs font-semibold text-slate-400">SELECT DRIFT SCENARIO</label>
            <div className="grid grid-cols-1 gap-2">
              {scenarios.map((sc) => {
                const Icon = sc.icon;
                const isSelected = scenario === sc.id;
                return (
                  <div
                    key={sc.id}
                    onClick={() => {
                      setScenario(sc.id);
                      setCount(sc.defaultCount);
                    }}
                    className={`p-3 rounded-lg border transition cursor-pointer flex items-start gap-3 ${
                      isSelected
                        ? 'bg-blue-600/15 border-blue-500 text-white'
                        : 'bg-sre-850/60 border-sre-800 text-slate-300 hover:bg-sre-850'
                    }`}
                  >
                    <Icon className={`w-5 h-5 mt-0.5 shrink-0 ${sc.color}`} />
                    <div className="flex-1">
                      <div className="font-medium text-xs text-white">{sc.title}</div>
                      <div className="text-[11px] text-slate-400 mt-0.5">{sc.description}</div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          <div>
            <label className="text-xs font-semibold text-slate-400 block mb-1">
              NUMBER OF TRACES TO INGEST
            </label>
            <div className="flex items-center gap-3">
              <input
                type="range"
                min="5"
                max="100"
                step="5"
                value={count}
                onChange={(e) => setCount(parseInt(e.target.value))}
                className="flex-1 accent-blue-500 cursor-pointer"
              />
              <span className="font-mono text-sm font-bold text-white w-10 text-right">
                {count}
              </span>
            </div>
          </div>

          <div className="pt-3 border-t border-sre-800 flex items-center justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              className="px-3 py-1.5 rounded-lg border border-sre-700 text-slate-300 text-xs hover:bg-sre-800 transition cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="flex items-center gap-1.5 px-4 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold transition cursor-pointer disabled:opacity-50"
            >
              {loading && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
              <span>{loading ? 'Injecting Traces...' : 'Inject Traffic'}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
