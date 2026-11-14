import React from 'react';
import { ShieldAlert, Zap, RefreshCw, Radio } from 'lucide-react';
import { WorkflowSummary } from '../types';

interface HeaderProps {
  workflows: WorkflowSummary[];
  selectedWorkflowId: string;
  onSelectWorkflow: (id: string) => void;
  sseConnected: boolean;
  onOpenSimulate: () => void;
  onOpenRecalibrate: () => void;
}

export const Header: React.FC<HeaderProps> = ({
  workflows,
  selectedWorkflowId,
  onSelectWorkflow,
  sseConnected,
  onOpenSimulate,
  onOpenRecalibrate,
}) => {
  return (
    <header className="border-b border-sre-800 bg-sre-900/80 backdrop-blur sticky top-0 z-30 px-6 py-3.5 flex flex-wrap items-center justify-between gap-4">
      <div className="flex items-center gap-3">
        <div className="flex items-center justify-center w-9 h-9 rounded-lg bg-blue-600/20 border border-blue-500/40 text-blue-400">
          <ShieldAlert className="w-5 h-5" />
        </div>
        <div>
          <div className="flex items-center gap-2">
            <span className="font-bold tracking-tight text-lg text-white">driftguard</span>
            <span className="text-[10px] font-mono uppercase px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-400 border border-blue-500/20">
              Sentinel v1.0
            </span>
          </div>
          <p className="text-xs text-slate-400 hidden sm:block">
            Autonomous Reasoning Divergence & Telemetry Sentinel
          </p>
        </div>
      </div>

      <div className="flex items-center gap-3">
        {/* Workflow Select */}
        <div className="flex items-center gap-2 bg-sre-850 border border-sre-700/60 rounded-lg px-2.5 py-1.5">
          <span className="text-xs text-slate-400 font-mono">Agent:</span>
          <select
            value={selectedWorkflowId}
            onChange={(e) => onSelectWorkflow(e.target.value)}
            className="bg-transparent text-sm text-slate-100 font-medium focus:outline-none cursor-pointer"
          >
            {workflows.map((wf) => (
              <option key={wf.id} value={wf.id} className="bg-sre-900 text-slate-100">
                {wf.name} ({wf.id})
              </option>
            ))}
          </select>
        </div>

        {/* SSE Stream Status */}
        <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border border-sre-800 bg-sre-850/60 text-xs">
          <Radio className={`w-3.5 h-3.5 ${sseConnected ? 'text-emerald-400 animate-pulse' : 'text-slate-500'}`} />
          <span className={`font-mono ${sseConnected ? 'text-emerald-400' : 'text-slate-400'}`}>
            {sseConnected ? 'LIVE SSE' : 'OFFLINE'}
          </span>
        </div>

        {/* Action Buttons */}
        <button
          onClick={onOpenSimulate}
          className="flex items-center gap-1.5 bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold px-3 py-2 rounded-lg transition shadow-sm cursor-pointer"
          title="Inject synthetic traffic or drift scenario"
        >
          <Zap className="w-3.5 h-3.5" />
          <span>Simulate Drift</span>
        </button>

        <button
          onClick={onOpenRecalibrate}
          className="flex items-center gap-1.5 bg-sre-800 hover:bg-sre-700 border border-sre-700 text-slate-200 text-xs font-medium px-3 py-2 rounded-lg transition cursor-pointer"
          title="Recalibrate Golden Baseline profile from recent traces"
        >
          <RefreshCw className="w-3.5 h-3.5" />
          <span>Recalibrate Baseline</span>
        </button>
      </div>
    </header>
  );
};
