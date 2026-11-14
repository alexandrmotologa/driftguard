import React, { useState } from 'react';
import { RefreshCw, X, AlertTriangle, Loader2 } from 'lucide-react';
import { Baseline } from '../types';

interface RecalibrateModalProps {
  workflowId: string;
  baseline?: Baseline;
  isOpen: boolean;
  onClose: () => void;
  onRecalibrate: () => Promise<void>;
}

export const RecalibrateModal: React.FC<RecalibrateModalProps> = ({
  workflowId,
  baseline,
  isOpen,
  onClose,
  onRecalibrate,
}) => {
  const [loading, setLoading] = useState<boolean>(false);

  if (!isOpen) return null;

  const handleConfirm = async () => {
    setLoading(true);
    try {
      await onRecalibrate();
      onClose();
    } catch (err) {
      console.error('Recalibration failed:', err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-sre-900 border border-sre-700 rounded-xl max-w-md w-full shadow-2xl overflow-hidden">
        <div className="flex items-center justify-between p-4 border-b border-sre-800">
          <div className="flex items-center gap-2">
            <RefreshCw className="w-5 h-5 text-blue-400" />
            <h3 className="font-semibold text-white">Recalibrate Golden Baseline</h3>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-sre-800 transition cursor-pointer"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-5 space-y-4 text-xs">
          <div className="p-3 rounded-lg bg-amber-950/30 border border-amber-800/60 flex items-start gap-2.5">
            <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
            <p className="text-amber-200">
              Recalibrating establishes a new statistical ground truth profile using the most recent 100 verified traces. Any current alerts will clear against this updated profile.
            </p>
          </div>

          <div className="space-y-2 font-mono">
            <div className="flex justify-between py-1 border-b border-sre-800 text-slate-300">
              <span className="text-slate-500">Target Workflow:</span>
              <span className="text-white font-bold">{workflowId}</span>
            </div>
            <div className="flex justify-between py-1 border-b border-sre-800 text-slate-300">
              <span className="text-slate-500">Current Baseline Mean:</span>
              <span className="text-white">{baseline?.expected_steps_mean.toFixed(2) || 'N/A'} steps</span>
            </div>
            <div className="flex justify-between py-1 border-b border-sre-800 text-slate-300">
              <span className="text-slate-500">Max Allowed Wasserstein:</span>
              <span className="text-white">{baseline?.max_allowed_wasserstein_distance.toFixed(1) || '2.0'}</span>
            </div>
            <div className="flex justify-between py-1 border-b border-sre-800 text-slate-300">
              <span className="text-slate-500">Last Profile Update:</span>
              <span className="text-white">
                {baseline?.updated_at ? new Date(baseline.updated_at).toLocaleString() : 'Default'}
              </span>
            </div>
          </div>

          <div className="pt-3 border-t border-sre-800 flex items-center justify-end gap-3 font-sans">
            <button
              type="button"
              onClick={onClose}
              className="px-3 py-1.5 rounded-lg border border-sre-700 text-slate-300 text-xs hover:bg-sre-800 transition cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="button"
              disabled={loading}
              onClick={handleConfirm}
              className="flex items-center gap-1.5 px-4 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold transition cursor-pointer disabled:opacity-50"
            >
              {loading && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
              <span>{loading ? 'Recalibrating...' : 'Confirm Recalibration'}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
