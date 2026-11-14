import React from 'react';
import { ShieldAlert, Zap, RefreshCw, Radio, Terminal, Power } from 'lucide-react';
import { WorkflowSummary, CircuitBreakerStatus } from '../types';

interface HeaderProps {
  workflows: WorkflowSummary[];
  selectedWorkflowId: string;
  onSelectWorkflow: (id: string) => void;
  sseConnected: boolean;
  circuitBreaker?: CircuitBreakerStatus;
  onResetCircuitBreaker: () => void;
  onOpenSimulate: () => void;
  onOpenRecalibrate: () => void;
  onOpenSandbox: () => void;
}

export const Header: React.FC<HeaderProps> = ({
  workflows,
  selectedWorkflowId,
  onSelectWorkflow,
  sseConnected,
  circuitBreaker,
  onResetCircuitBreaker,
  onOpenSimulate,
  onOpenRecalibrate,
  onOpenSandbox,
}) => {
  const isBreakerOpen = circuitBreaker?.state === 'OPEN';

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

      <div className="flex flex-wrap items-center gap-2 sm:gap-3">
        {/* Workflow Select */}
        <div className="flex items-center gap-1.5 sm:gap-2 bg-sre-850 border border-sre-700/60 rounded-lg px-2 sm:px-2.5 py-1.5">
          <span className="text-xs text-slate-400 font-mono">Agent:</span>
          <select
            value={selectedWorkflowId}
            onChange={(e) => onSelectWorkflow(e.target.value)}
            className="bg-transparent text-xs sm:text-sm text-slate-100 font-medium focus:outline-none cursor-pointer max-w-[120px] sm:max-w-none truncate"
          >
            {workflows.map((wf) => (
              <option key={wf.id} value={wf.id} className="bg-sre-900 text-slate-100">
                {wf.name} ({wf.id})
              </option>
            ))}
          </select>
        </div>

        {/* Circuit Breaker Status */}
        {circuitBreaker && (
          <div
            className={`flex items-center gap-1.5 px-2 sm:px-2.5 py-1.5 rounded-lg border text-xs font-mono transition ${
              isBreakerOpen
                ? 'bg-rose-950/50 border-rose-700 text-rose-300'
                : 'bg-sre-850 border-sre-700/60 text-slate-300'
            }`}
          >
            <span
              className={`w-2 h-2 rounded-full ${
                isBreakerOpen ? 'bg-rose-500 animate-pulse' : 'bg-emerald-400'
              }`}
            />
            <span>CB: {circuitBreaker.state}</span>
            {isBreakerOpen && (
              <button
                onClick={onResetCircuitBreaker}
                className="ml-1 text-[10px] px-1.5 py-0.5 rounded bg-rose-600 hover:bg-rose-500 text-white font-semibold transition"
                title="Reset Circuit Breaker"
              >
                Reset
              </button>
            )}
          </div>
        )}

        {/* SSE Stream Status */}
        <div className="flex items-center gap-1.5 px-2 sm:px-2.5 py-1.5 rounded-lg border border-sre-800 bg-sre-850/60 text-xs">
          <Radio className={`w-3.5 h-3.5 ${sseConnected ? 'text-emerald-400 animate-pulse' : 'text-slate-500'}`} />
          <span className={`font-mono ${sseConnected ? 'text-emerald-400' : 'text-slate-400'}`}>
            {sseConnected ? 'LIVE SSE' : 'OFFLINE'}
          </span>
        </div>

        {/* Trace Sandbox Button */}
        <button
          onClick={onOpenSandbox}
          className="flex items-center gap-1.5 bg-sre-800 hover:bg-sre-700 border border-sre-700 text-slate-200 text-xs font-medium px-2.5 sm:px-3 py-1.5 sm:py-2 rounded-lg transition cursor-pointer"
          title="Open Ad-Hoc Trace Testing Sandbox"
        >
          <Terminal className="w-3.5 h-3.5 text-emerald-400" />
          <span>Sandbox</span>
        </button>

        {/* Action Buttons */}
        <button
          onClick={onOpenSimulate}
          className="flex items-center gap-1.5 bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold px-2.5 sm:px-3 py-1.5 sm:py-2 rounded-lg transition shadow-sm cursor-pointer"
          title="Inject synthetic traffic or drift scenario"
        >
          <Zap className="w-3.5 h-3.5" />
          <span>Simulate<span className="hidden md:inline"> Drift</span></span>
        </button>

        <button
          onClick={onOpenRecalibrate}
          className="flex items-center gap-1.5 bg-sre-800 hover:bg-sre-700 border border-sre-700 text-slate-200 text-xs font-medium px-2.5 sm:px-3 py-1.5 sm:py-2 rounded-lg transition cursor-pointer"
          title="Recalibrate Golden Baseline profile from recent traces"
        >
          <RefreshCw className="w-3.5 h-3.5" />
          <span>Recalibrate<span className="hidden md:inline"> Baseline</span></span>
        </button>
      </div>
    </header>
  );
};
