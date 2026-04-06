import { useState, useEffect } from 'react';
import type { ContentPrompt, PromptAction } from '../types';

interface Props {
  prompt: ContentPrompt;
  onResolve: (promptId: string, action: PromptAction, filePaths: string[]) => void;
}

export function ContentPromptModal({ prompt, onResolve }: Props) {
  const [countdown, setCountdown] = useState(30);

  useEffect(() => {
    setCountdown(30);
    const timer = setInterval(() => {
      setCountdown(prev => {
        if (prev <= 1) {
          clearInterval(timer);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(timer);
  }, [prompt.id]);

  const handle = (action: PromptAction) => {
    onResolve(prompt.id, action, prompt.file_paths);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70">
      <div className="bg-[#1a1a2e] border border-gray-700 rounded-lg shadow-2xl max-w-lg w-full mx-4">
        {/* Header */}
        <div className="px-5 py-3 border-b border-gray-700 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-yellow-300">Content Review Required</h2>
          <span className="text-xs text-gray-500">auto-block in {countdown}s</span>
        </div>

        {/* Body */}
        <div className="px-5 py-4 space-y-3">
          <div>
            <span className="text-xs text-gray-500">Matched keyword</span>
            <div className="mt-1 px-2 py-1 bg-yellow-900/30 border border-yellow-800/50 rounded">
              <span className="font-mono text-sm text-yellow-300">{prompt.matched_keyword}</span>
            </div>
          </div>

          <div>
            <span className="text-xs text-gray-500">Target</span>
            <div className="mt-1 font-mono text-xs text-gray-300 truncate">{prompt.url}</div>
          </div>

          <div>
            <span className="text-xs text-gray-500">Files in request ({prompt.file_paths.length})</span>
            <div className="mt-1 space-y-1 max-h-32 overflow-auto">
              {prompt.file_paths.map(fp => (
                <div key={fp} className="font-mono text-xs text-gray-300 bg-[#0f0f1a] rounded px-2 py-1">
                  {fp}
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="px-5 py-3 border-t border-gray-700 grid grid-cols-2 gap-2">
          <button
            onClick={() => handle('allow_once')}
            className="px-3 py-2 rounded text-xs font-medium bg-green-900/40 text-green-300 hover:bg-green-900/60 transition-colors"
          >
            Allow Once
          </button>
          <button
            onClick={() => handle('allow_always')}
            className="px-3 py-2 rounded text-xs font-medium bg-emerald-900/40 text-emerald-300 hover:bg-emerald-900/60 transition-colors"
          >
            Allow Always
          </button>
          <button
            onClick={() => handle('block_once')}
            className="px-3 py-2 rounded text-xs font-medium bg-red-900/40 text-red-300 hover:bg-red-900/60 transition-colors"
          >
            Block Once
          </button>
          <button
            onClick={() => handle('block_always')}
            className="px-3 py-2 rounded text-xs font-medium bg-red-950/50 text-red-400 hover:bg-red-950/70 transition-colors"
          >
            Block Always
          </button>
        </div>
      </div>
    </div>
  );
}