import React from 'react';
import { Incident } from '../types';
import { AlertOctagon, AlertTriangle, CheckCircle2, ArrowRight } from 'lucide-react';

interface IncidentFeedProps {
  incidents: Incident[];
  onAcknowledge: (id: string) => void;
}

export const IncidentFeed: React.FC<IncidentFeedProps> = ({
  incidents,
  onAcknowledge,
}) => {
  return (
    <div className="bg-sre-900 border border-sre-800 rounded-xl p-5 flex flex-col h-full">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <AlertOctagon className="w-4 h-4 text-rose-400" />
          <h3 className="font-semibold text-sm text-white">Anomalies & SRE Incidents</h3>
        </div>
        <span className="text-xs font-mono text-slate-400">
          {incidents.filter((i) => !i.acknowledged).length} active alerts
        </span>
      </div>

      {incidents.length === 0 ? (
        <div className="flex-1 flex flex-col items-center justify-center p-8 text-center border border-dashed border-sre-800 rounded-lg">
          <CheckCircle2 className="w-8 h-8 text-emerald-500/80 mb-2" />
          <p className="text-sm text-slate-300 font-medium">All Sentinels Clear</p>
          <p className="text-xs text-slate-400 mt-1">
            Zero behavioral anomalies or guardrail breaches detected in recent runs.
          </p>
        </div>
      ) : (
        <div className="space-y-3 overflow-y-auto max-h-[360px] pr-1">
          {incidents.map((inc) => (
            <div
              key={inc.id}
              className={`p-3.5 rounded-lg border transition ${
                inc.acknowledged
                  ? 'bg-sre-850/40 border-sre-800/60 opacity-60'
                  : inc.severity === 'CRITICAL'
                  ? 'bg-rose-950/20 border-rose-800/60'
                  : 'bg-amber-950/20 border-amber-800/60'
              }`}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="flex items-center gap-2">
                  {inc.severity === 'CRITICAL' ? (
                    <span className="flex items-center gap-1 text-[11px] font-bold font-mono px-2 py-0.5 rounded bg-rose-500/20 text-rose-400 border border-rose-500/30">
                      <AlertOctagon className="w-3 h-3" />
                      CRITICAL
                    </span>
                  ) : (
                    <span className="flex items-center gap-1 text-[11px] font-bold font-mono px-2 py-0.5 rounded bg-amber-500/20 text-amber-400 border border-amber-500/30">
                      <AlertTriangle className="w-3 h-3" />
                      WARNING
                    </span>
                  )}
                  <span className="text-xs font-mono text-slate-400 uppercase">
                    {inc.anomaly_type}
                  </span>
                </div>

                <div className="flex items-center gap-2">
                  <span className="text-[11px] text-slate-400 font-mono">
                    {new Date(inc.created_at).toLocaleTimeString()}
                  </span>
                  {!inc.acknowledged && (
                    <button
                      onClick={() => onAcknowledge(inc.id)}
                      className="text-[11px] font-medium bg-sre-800 hover:bg-sre-700 text-slate-300 hover:text-white px-2 py-0.5 rounded border border-sre-700 transition cursor-pointer"
                    >
                      Acknowledge
                    </button>
                  )}
                </div>
              </div>

              <p className="text-xs text-slate-200 mt-2 font-medium">
                {inc.summary}
              </p>

              {inc.offending_transitions && inc.offending_transitions.length > 0 && (
                <div className="mt-2.5 pt-2 border-t border-sre-800/80 flex flex-wrap items-center gap-1.5 text-[11px] font-mono">
                  <span className="text-slate-400">Offending:</span>
                  {inc.offending_transitions.map((edge, idx) => (
                    <span
                      key={idx}
                      className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-rose-950/60 text-rose-300 border border-rose-800/50"
                    >
                      <span>{edge.from || 'START'}</span>
                      <ArrowRight className="w-2.5 h-2.5 text-rose-400" />
                      <span>{edge.to || 'END'}</span>
                      {edge.count > 1 && (
                        <span className="text-slate-400 text-[10px]">({edge.count}x)</span>
                      )}
                    </span>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
