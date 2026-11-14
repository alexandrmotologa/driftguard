import React, { useState, useEffect, useCallback } from 'react';
import {
  WorkflowSummary,
  Baseline,
  EvaluationResult,
  Incident,
  Trace,
  TimelinePoint,
  TrajectoryDiff,
} from './types';
import { Header } from './components/Header';
import { OverviewCards } from './components/OverviewCards';
import { MarkovHeatmap } from './components/MarkovHeatmap';
import { ToolGraphDAG } from './components/ToolGraphDAG';
import { StepHistogram } from './components/StepHistogram';
import { DriftTimeline } from './components/DriftTimeline';
import { IncidentFeed } from './components/IncidentFeed';
import { TraceExplorer } from './components/TraceExplorer';
import { SimulateModal } from './components/SimulateModal';
import { RecalibrateModal } from './components/RecalibrateModal';
import { TraceDiffModal } from './components/TraceDiffModal';
import { SandboxModal } from './components/SandboxModal';
import { Grid, Share2 } from 'lucide-react';

export const App: React.FC = () => {
  const [workflows, setWorkflows] = useState<WorkflowSummary[]>([]);
  const [selectedWorkflowId, setSelectedWorkflowId] = useState<string>('');
  const [baseline, setBaseline] = useState<Baseline | undefined>(undefined);
  const [evaluation, setEvaluation] = useState<EvaluationResult | undefined>(undefined);
  const [timeline, setTimeline] = useState<TimelinePoint[]>([]);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [traces, setTraces] = useState<Trace[]>([]);
  const [sseConnected, setSseConnected] = useState<boolean>(false);

  // View state: Matrix table vs DAG graph
  const [topologyMode, setTopologyMode] = useState<'matrix' | 'dag'>('matrix');

  // Modals state
  const [isSimulateOpen, setIsSimulateOpen] = useState<boolean>(false);
  const [isRecalibrateOpen, setIsRecalibrateOpen] = useState<boolean>(false);
  const [isSandboxOpen, setIsSandboxOpen] = useState<boolean>(false);
  const [isDiffOpen, setIsDiffOpen] = useState<boolean>(false);
  const [selectedDiff, setSelectedDiff] = useState<TrajectoryDiff | null>(null);

  // Fetch all workflows
  const fetchWorkflows = useCallback(async () => {
    try {
      const res = await fetch('/api/v1/workflows');
      if (res.ok) {
        const data: WorkflowSummary[] = await res.json();
        setWorkflows(data);
        if (!selectedWorkflowId && data.length > 0) {
          setSelectedWorkflowId(data[0].id);
        }
      }
    } catch (err) {
      console.error('Failed to fetch workflows:', err);
    }
  }, [selectedWorkflowId]);

  // Fetch data for selected workflow
  const fetchWorkflowData = useCallback(async (id: string) => {
    if (!id) return;
    try {
      const [baseRes, metricsRes, incRes, traceRes, timelineRes] = await Promise.all([
        fetch(`/api/v1/workflows/${id}/baseline`),
        fetch(`/api/v1/workflows/${id}/metrics`),
        fetch(`/api/v1/workflows/${id}/incidents`),
        fetch(`/api/v1/workflows/${id}/traces?limit=30`),
        fetch(`/api/v1/workflows/${id}/evaluations/timeline`),
      ]);

      if (baseRes.ok) {
        const b = await baseRes.json();
        setBaseline(b);
      }
      if (metricsRes.ok) {
        const m = await metricsRes.json();
        setEvaluation(m);
      }
      if (incRes.ok) {
        const i = await incRes.json();
        setIncidents(i);
      }
      if (traceRes.ok) {
        const t = await traceRes.json();
        setTraces(t);
      }
      if (timelineRes.ok) {
        const tl = await timelineRes.json();
        setTimeline(tl);
      }
    } catch (err) {
      console.error(`Failed to fetch workflow ${id} data:`, err);
    }
  }, []);

  // Initial load
  useEffect(() => {
    fetchWorkflows();
  }, [fetchWorkflows]);

  // When selected workflow changes
  useEffect(() => {
    if (selectedWorkflowId) {
      fetchWorkflowData(selectedWorkflowId);
    }
  }, [selectedWorkflowId, fetchWorkflowData]);

  // Connect SSE EventSource
  useEffect(() => {
    const es = new EventSource('/api/v1/events');

    es.onopen = () => {
      setSseConnected(true);
    };

    es.onerror = () => {
      setSseConnected(false);
    };

    es.addEventListener('trace_ingested', () => {
      if (selectedWorkflowId) fetchWorkflowData(selectedWorkflowId);
      fetchWorkflows();
    });

    es.addEventListener('trace_batch_ingested', () => {
      if (selectedWorkflowId) fetchWorkflowData(selectedWorkflowId);
      fetchWorkflows();
    });

    es.addEventListener('evaluation_completed', (e) => {
      try {
        const payload = JSON.parse(e.data);
        const evalResult: EvaluationResult = payload.payload;
        if (evalResult.workflow_id === selectedWorkflowId) {
          setEvaluation(evalResult);
        }
        fetchWorkflows();
      } catch (err) {
        console.error('Error handling evaluation event:', err);
      }
    });

    es.addEventListener('incident_triggered', (e) => {
      try {
        const payload = JSON.parse(e.data);
        const inc: Incident = payload.payload;
        if (inc.workflow_id === selectedWorkflowId) {
          setIncidents((prev) => [inc, ...prev.filter((item) => item.id !== inc.id)]);
        }
        fetchWorkflows();
      } catch (err) {
        console.error('Error handling incident event:', err);
      }
    });

    es.addEventListener('incident_acknowledged', () => {
      if (selectedWorkflowId) fetchWorkflowData(selectedWorkflowId);
    });

    es.addEventListener('baseline_updated', () => {
      if (selectedWorkflowId) fetchWorkflowData(selectedWorkflowId);
    });

    es.addEventListener('traffic_simulated', () => {
      if (selectedWorkflowId) fetchWorkflowData(selectedWorkflowId);
      fetchWorkflows();
    });

    es.addEventListener('circuit_breaker_tripped', () => {
      fetchWorkflows();
    });

    es.addEventListener('circuit_breaker_reset', () => {
      fetchWorkflows();
    });

    return () => {
      es.close();
    };
  }, [selectedWorkflowId, fetchWorkflowData, fetchWorkflows]);

  // Handle acknowledge incident
  const handleAcknowledge = async (id: string) => {
    try {
      const res = await fetch(`/api/v1/incidents/${id}/ack`, { method: 'POST' });
      if (res.ok) {
        setIncidents((prev) =>
          prev.map((i) => (i.id === id ? { ...i, acknowledged: true } : i))
        );
      }
    } catch (err) {
      console.error('Failed to acknowledge incident:', err);
    }
  };

  // Handle Circuit Breaker Reset
  const handleResetCircuitBreaker = async () => {
    try {
      const res = await fetch(`/api/v1/workflows/${selectedWorkflowId}/circuit-breaker/reset`, {
        method: 'POST',
      });
      if (res.ok) {
        fetchWorkflows();
      }
    } catch (err) {
      console.error('Failed to reset circuit breaker:', err);
    }
  };

  // Handle Inspect Diff for a trace
  const handleInspectDiff = async (traceId: string) => {
    try {
      const res = await fetch(`/api/v1/workflows/${selectedWorkflowId}/traces/${traceId}/diff`);
      if (res.ok) {
        const diffData: TrajectoryDiff = await res.json();
        setSelectedDiff(diffData);
        setIsDiffOpen(true);
      }
    } catch (err) {
      console.error('Failed to inspect diff:', err);
    }
  };

  // Handle Simulate Traffic
  const handleSimulate = async (scenario: string, count: number) => {
    const res = await fetch('/api/v1/simulate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        workflow_id: selectedWorkflowId,
        scenario,
        count,
      }),
    });
    if (!res.ok) {
      throw new Error('Simulation failed');
    }
  };

  // Handle Recalibrate
  const handleRecalibrate = async () => {
    const res = await fetch(`/api/v1/workflows/${selectedWorkflowId}/recalibrate`, {
      method: 'POST',
    });
    if (!res.ok) {
      throw new Error('Recalibration failed');
    }
  };

  const currentSummary = workflows.find((w) => w.id === selectedWorkflowId);

  return (
    <div className="min-h-screen bg-sre-950 text-slate-100 flex flex-col">
      <Header
        workflows={workflows}
        selectedWorkflowId={selectedWorkflowId}
        onSelectWorkflow={setSelectedWorkflowId}
        sseConnected={sseConnected}
        circuitBreaker={currentSummary?.circuit_breaker}
        onResetCircuitBreaker={handleResetCircuitBreaker}
        onOpenSimulate={() => setIsSimulateOpen(true)}
        onOpenRecalibrate={() => setIsRecalibrateOpen(true)}
        onOpenSandbox={() => setIsSandboxOpen(true)}
      />

      <main className="flex-1 p-6 max-w-7xl w-full mx-auto space-y-6">
        {/* KPI Overview Cards */}
        <OverviewCards
          evaluation={evaluation}
          baseline={baseline}
          totalTraces={currentSummary?.trace_count ?? traces.length}
        />

        {/* Historical Divergence Timeline Chart */}
        <DriftTimeline
          timeline={timeline}
          threshold={baseline?.max_allowed_wasserstein_distance ?? 2.0}
        />

        {/* Middle Section: Heatmap/Graph & Histogram (Left) + Incidents (Right) */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
            {/* View Mode Header */}
            <div className="flex items-center justify-between bg-sre-900 border border-sre-800 rounded-xl px-4 py-2 text-xs font-mono">
              <span className="text-slate-400">Tool State Representation:</span>
              <div className="flex items-center gap-1 bg-sre-850 p-1 rounded-lg border border-sre-700/60">
                <button
                  onClick={() => setTopologyMode('matrix')}
                  className={`flex items-center gap-1.5 px-3 py-1 rounded transition cursor-pointer ${
                    topologyMode === 'matrix'
                      ? 'bg-blue-600 text-white font-medium'
                      : 'text-slate-400 hover:text-slate-200'
                  }`}
                >
                  <Grid className="w-3.5 h-3.5" />
                  <span>Markov Table</span>
                </button>
                <button
                  onClick={() => setTopologyMode('dag')}
                  className={`flex items-center gap-1.5 px-3 py-1 rounded transition cursor-pointer ${
                    topologyMode === 'dag'
                      ? 'bg-purple-600 text-white font-medium'
                      : 'text-slate-400 hover:text-slate-200'
                  }`}
                >
                  <Share2 className="w-3.5 h-3.5" />
                  <span>Directed DAG</span>
                </button>
              </div>
            </div>

            {topologyMode === 'matrix' ? (
              <MarkovHeatmap
                currentMatrix={evaluation?.current_matrix}
                baselineMatrix={baseline?.markov_matrix}
                perturbation={evaluation?.matrix_deviation}
              />
            ) : (
              <ToolGraphDAG
                matrix={evaluation?.current_matrix}
                perturbation={evaluation?.matrix_deviation}
              />
            )}

            <StepHistogram
              histogram={evaluation?.step_histogram}
              wassersteinDistance={evaluation?.wasserstein_distance ?? 0}
            />
          </div>

          <div className="lg:col-span-1">
            <IncidentFeed
              incidents={incidents}
              onAcknowledge={handleAcknowledge}
              onInspectDiff={handleInspectDiff}
            />
          </div>
        </div>

        {/* Bottom Section: Trace Stream Explorer */}
        <TraceExplorer
          traces={traces}
          onInspectDiff={handleInspectDiff}
        />
      </main>

      {/* Modals */}
      <SimulateModal
        workflowId={selectedWorkflowId}
        isOpen={isSimulateOpen}
        onClose={() => setIsSimulateOpen(false)}
        onSimulate={handleSimulate}
      />

      <RecalibrateModal
        workflowId={selectedWorkflowId}
        baseline={baseline}
        isOpen={isRecalibrateOpen}
        onClose={() => setIsRecalibrateOpen(false)}
        onRecalibrate={handleRecalibrate}
      />

      <TraceDiffModal
        diff={selectedDiff}
        isOpen={isDiffOpen}
        onClose={() => {
          setIsDiffOpen(false);
          setSelectedDiff(null);
        }}
      />

      <SandboxModal
        workflowId={selectedWorkflowId}
        isOpen={isSandboxOpen}
        onClose={() => setIsSandboxOpen(false)}
      />
    </div>
  );
};
