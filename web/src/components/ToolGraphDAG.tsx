import React, { useState } from 'react';
import { MarkovMatrix, MatrixPerturbation } from '../types';
import { Share2, Repeat, AlertTriangle } from 'lucide-react';

interface ToolGraphDAGProps {
  matrix?: MarkovMatrix;
  perturbation?: MatrixPerturbation;
}

export const ToolGraphDAG: React.FC<ToolGraphDAGProps> = ({
  matrix,
  perturbation,
}) => {
  const [selectedNode, setSelectedNode] = useState<string | null>(null);

  if (!matrix || !matrix.states || matrix.states.length === 0) {
    return (
      <div className="bg-sre-900 border border-sre-800 rounded-xl p-5 text-center text-slate-400 text-xs font-mono">
        No state transition topology available.
      </div>
    );
  }

  // Filter and order states
  const states = [...matrix.states].sort((a, b) => {
    if (a === '__START__') return -1;
    if (b === '__START__') return 1;
    if (a === '__END__') return 1;
    if (b === '__END__') return -1;
    return a.localeCompare(b);
  });

  // Calculate circular layout positions
  const width = 640;
  const height = 360;
  const centerX = width / 2;
  const centerY = height / 2;
  const radius = 135;

  const nodePositions: Record<string, { x: number; y: number }> = {};
  const toolStates = states.filter((s) => s !== '__START__' && s !== '__END__');

  // Place START on left, END on right, others in a circle
  nodePositions['__START__'] = { x: 60, y: centerY };
  nodePositions['__END__'] = { x: width - 60, y: centerY };

  const angleStep = (2 * Math.PI) / (toolStates.length || 1);
  toolStates.forEach((st, idx) => {
    const angle = idx * angleStep - Math.PI / 2;
    nodePositions[st] = {
      x: centerX + radius * Math.cos(angle),
      y: centerY + radius * Math.sin(angle),
    };
  });

  // Extract non-trivial edges (probability >= 0.05)
  interface GraphEdge {
    from: string;
    to: string;
    prob: number;
    count: number;
    isLoop: boolean;
    isNovel: boolean;
  }

  const edges: GraphEdge[] = [];
  const novelSet = new Set<string>();
  perturbation?.novel_transitions?.forEach((nt) => {
    novelSet.add(`${nt.from}->${nt.to}`);
  });

  states.forEach((from) => {
    states.forEach((to) => {
      const prob = matrix.probabilities?.[from]?.[to] ?? 0;
      const count = matrix.raw_counts?.[from]?.[to] ?? 0;
      const isNovel = novelSet.has(`${from}->${to}`);

      if (prob >= 0.06 || count > 0 || isNovel) {
        edges.push({
          from,
          to,
          prob,
          count,
          isLoop: from === to,
          isNovel,
        });
      }
    });
  });

  return (
    <div className="bg-sre-900 border border-sre-800 rounded-xl p-5">
      <div className="flex items-center justify-between mb-3">
        <div>
          <div className="flex items-center gap-2">
            <Share2 className="w-4 h-4 text-purple-400" />
            <h3 className="font-semibold text-sm text-white">
              Directed Tool Transition Topology (Markov Chain DAG)
            </h3>
          </div>
          <p className="text-xs text-slate-400 mt-0.5">
            Node-link representation of reasoning flow paths and state loop traps
          </p>
        </div>

        <div className="flex items-center gap-3 text-xs font-mono">
          <div className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-blue-500 inline-block" />
            <span className="text-slate-300">Tool Node</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-rose-500 inline-block" />
            <span className="text-rose-400">Loop / Anomaly</span>
          </div>
        </div>
      </div>

      <div className="relative w-full overflow-hidden bg-sre-950/60 rounded-lg border border-sre-800/80">
        <svg viewBox={`0 0 ${width} ${height}`} className="w-full h-80 select-none">
          <defs>
            <marker
              id="arrow"
              viewBox="0 0 10 10"
              refX="18"
              refY="5"
              markerWidth="6"
              markerHeight="6"
              orient="auto-start-reverse"
            >
              <path d="M 0 1 L 10 5 L 0 9 z" fill="#64748b" />
            </marker>
            <marker
              id="arrow-active"
              viewBox="0 0 10 10"
              refX="18"
              refY="5"
              markerWidth="6"
              markerHeight="6"
              orient="auto-start-reverse"
            >
              <path d="M 0 1 L 10 5 L 0 9 z" fill="#3b82f6" />
            </marker>
            <marker
              id="arrow-novel"
              viewBox="0 0 10 10"
              refX="18"
              refY="5"
              markerWidth="6"
              markerHeight="6"
              orient="auto-start-reverse"
            >
              <path d="M 0 1 L 10 5 L 0 9 z" fill="#ef4444" />
            </marker>
          </defs>

          {/* Render Normal Links */}
          {edges
            .filter((e) => !e.isLoop)
            .map((edge, idx) => {
              const src = nodePositions[edge.from];
              const dst = nodePositions[edge.to];
              if (!src || !dst) return null;

              const isHighlighted =
                selectedNode === null || selectedNode === edge.from || selectedNode === edge.to;
              const isNovel = edge.isNovel;

              // Arc curvature calculation
              const dx = dst.x - src.x;
              const dy = dst.y - src.y;
              const dr = Math.sqrt(dx * dx + dy * dy) * 1.2;

              return (
                <g key={idx} opacity={isHighlighted ? 1 : 0.2}>
                  <path
                    d={`M ${src.x} ${src.y} A ${dr} ${dr} 0 0,1 ${dst.x} ${dst.y}`}
                    fill="none"
                    stroke={isNovel ? '#ef4444' : isHighlighted && selectedNode ? '#3b82f6' : '#475569'}
                    strokeWidth={Math.max(1.2, Math.min(4, edge.prob * 5))}
                    strokeDasharray={isNovel ? '4 3' : undefined}
                    markerEnd={isNovel ? 'url(#arrow-novel)' : isHighlighted && selectedNode ? 'url(#arrow-active)' : 'url(#arrow)'}
                    className="transition-all"
                  />
                </g>
              );
            })}

          {/* Render Self-Loops */}
          {edges
            .filter((e) => e.isLoop && e.from !== '__START__' && e.from !== '__END__')
            .map((edge, idx) => {
              const pos = nodePositions[edge.from];
              if (!pos) return null;

              const isCriticalLoop = edge.prob > 0.20;
              const loopRadius = 20;

              return (
                <g key={`loop-${idx}`}>
                  <circle
                    cx={pos.x}
                    cy={pos.y - 25}
                    r={loopRadius}
                    fill="none"
                    stroke={isCriticalLoop ? '#ef4444' : '#f59e0b'}
                    strokeWidth={isCriticalLoop ? 2.5 : 1.5}
                    strokeDasharray={isCriticalLoop ? undefined : '3 2'}
                  />
                  <text
                    x={pos.x}
                    y={pos.y - 30}
                    textAnchor="middle"
                    fill={isCriticalLoop ? '#f87171' : '#fbbf24'}
                    className="text-[10px] font-mono font-bold"
                  >
                    {(edge.prob * 100).toFixed(0)}%
                  </text>
                </g>
              );
            })}

          {/* Render State Nodes */}
          {states.map((st) => {
            const pos = nodePositions[st];
            if (!pos) return null;

            const isSelected = selectedNode === st;
            const isEndpoint = st === '__START__' || st === '__END__';
            const displayName = st.replace('__', '');

            return (
              <g
                key={st}
                className="cursor-pointer transition group"
                onClick={() => setSelectedNode(isSelected ? null : st)}
              >
                <circle
                  cx={pos.x}
                  cy={pos.y}
                  r={isEndpoint ? 18 : 22}
                  fill={isEndpoint ? '#1e293b' : isSelected ? '#2563eb' : '#0f172a'}
                  stroke={isSelected ? '#60a5fa' : isEndpoint ? '#475569' : '#3b82f6'}
                  strokeWidth={isSelected ? 3 : 2}
                  className="transition-all"
                />
                <text
                  x={pos.x}
                  y={pos.y + 4}
                  textAnchor="middle"
                  fill="#f8fafc"
                  className="text-[10px] font-mono font-semibold pointer-events-none"
                >
                  {isEndpoint ? displayName : displayName.slice(0, 4)}
                </text>
                {/* Node Label Below */}
                <text
                  x={pos.x}
                  y={pos.y + 36}
                  textAnchor="middle"
                  fill={isSelected ? '#93c5fd' : '#cbd5e1'}
                  className="text-[11px] font-mono font-medium pointer-events-none"
                >
                  {displayName}
                </text>
              </g>
            );
          })}
        </svg>

        <div className="p-3 border-t border-sre-800/80 bg-sre-900/40 flex items-center justify-between text-xs text-slate-400 font-mono">
          <span>Click any node to isolate inbound & outbound execution paths</span>
          {selectedNode && (
            <button
              onClick={() => setSelectedNode(null)}
              className="text-blue-400 hover:text-blue-300 underline"
            >
              Reset Selection
            </button>
          )}
        </div>
      </div>
    </div>
  );
};
