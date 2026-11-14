import React, { useState, useEffect, useCallback } from 'react';
import { WorkflowSummary, Baseline, EvaluationResult, Incident, Trace } from './types';
import { Header } from './components/Header';
import { OverviewCards } from './components/OverviewCards';
import { MarkovHeatmap } from './components/MarkovHeatmap';
import { StepHistogram } from './components/StepHistogram';
import { IncidentFeed } from './components/IncidentFeed';
import { TraceExplorer } from './components/TraceExplorer';
import { SimulateModal } from './components/SimulateModal';
import { RecalibrateModal } from './components/RecalibrateModal';

export const App: React.FC = () => {
  const [workflows, setWorkflows] = useState<WorkflowSummary[]>([]);
  const [selectedWorkflowId, setSelectedWorkflowId] = useState<string>('');
  const [baseline, setBaseline] = useState<Baseline | undefined>(undefined);
  const [evaluation, setEvaluation] = useState<EvaluationResult | undefined>(undefined);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [traces, setTraces] = useState<Trace[]>([]);
  const [sseConnected, setSseConnected] = useState<boolean>(false);

  // Modals state
  const [isSimulateOpen, setIsSimulateOpen] = useState<boolean>(false);
  const [isRecalibrateOpen, setIsRecalibrateOpen] = useState<boolean>(false);

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
      const [baseRes, metricsRes, incRes, traceRes] = await Promise.all([
        fetch(`/api/v1/workflows/${id}/baseline`),
        fetch(`/api/v1/workflows/${id}/metrics`),
        fetch(`/api/v1/workflows/${id}/incidents`),
        fetch(`/api/v1/workflows/${id}/traces?limit=30`),
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
        onOpenSimulate={() => setIsSimulateOpen(true)}
        onOpenRecalibrate={() => setIsRecalibrateOpen(true)}
      />

      <main className="flex-1 p-6 max-w-7xl w-full mx-auto space-y-6">
        {/* KPI Overview Cards */}
        <OverviewCards
          evaluation={evaluation}
          baseline={baseline}
          totalTraces={currentSummary?.trace_count ?? traces.length}
        />

        {/* Middle Section: Heatmap & Histogram (Left) + Incidents (Right) */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
            <MarkovHeatmap
              currentMatrix={evaluation?.current_matrix}
              baselineMatrix={baseline?.markov_matrix}
              perturbation={evaluation?.matrix_deviation}
            />

            <StepHistogram
              histogram={evaluation?.step_histogram}
              wassersteinDistance={evaluation?.wasserstein_distance ?? 0}
            />
          </div>

          <div className="lg:col-span-1">
            <IncidentFeed
              incidents={incidents}
              onAcknowledge={handleAcknowledge}
            />
          </div>
        </div>

        {/* Bottom Section: Trace Stream Explorer */}
        <TraceExplorer traces={traces} />
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
    </div>
  );
};
