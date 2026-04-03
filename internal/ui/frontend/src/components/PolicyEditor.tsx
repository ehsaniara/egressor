import { useState } from 'react';

interface Props {
  patterns: string[];
  onAdd: (pattern: string) => void;
  onRemove: (pattern: string) => void;
  allowedDirs: string[];
  onAddDir: (dir: string) => void;
  onRemoveDir: (dir: string) => void;
  onSave: () => void;
}

export function PolicyEditor({ patterns, onAdd, onRemove, allowedDirs, onAddDir, onRemoveDir, onSave }: Props) {
  const [patternInput, setPatternInput] = useState('');
  const [dirInput, setDirInput] = useState('');
  const [saved, setSaved] = useState(false);

  const handleAddPattern = () => {
    const trimmed = patternInput.trim();
    if (trimmed && !patterns.includes(trimmed)) {
      onAdd(trimmed);
      setPatternInput('');
    }
  };

  const handleAddDir = () => {
    const trimmed = dirInput.trim();
    if (trimmed && !allowedDirs.includes(trimmed)) {
      onAddDir(trimmed);
      setDirInput('');
    }
  };

  const handleSave = () => {
    onSave();
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  };

  return (
    <div className="p-4 space-y-6">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-gray-300">Policy Rules</h3>
        <button
          onClick={handleSave}
          className={`px-3 py-1 rounded text-xs font-medium transition-colors ${
            saved
              ? 'bg-green-900/50 text-green-300'
              : 'bg-gray-700/50 text-gray-300 hover:bg-gray-700/70'
          }`}
        >
          {saved ? 'Saved' : 'Save to config'}
        </button>
      </div>

      {/* Allowed Directories */}
      <div className="space-y-3">
        <div>
          <h4 className="text-xs font-medium text-gray-400 mb-1">Allowed Directories</h4>
          <p className="text-xs text-gray-600 mb-2">
            Files outside these directories will be blocked. Leave empty to allow all.
          </p>
        </div>

        <div className="flex gap-2">
          <input
            value={dirInput}
            onChange={(e) => setDirInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleAddDir()}
            placeholder="e.g. ~/Projects/my-app"
            className="flex-1 bg-[#1a1a2e] border border-gray-700 rounded px-3 py-1.5 text-xs text-gray-300 font-mono placeholder-gray-600 focus:outline-none focus:border-blue-600"
          />
          <button
            onClick={handleAddDir}
            className="px-3 py-1.5 bg-blue-900/50 text-blue-300 rounded text-xs font-medium hover:bg-blue-900/70"
          >
            Add
          </button>
        </div>

        <div className="space-y-1">
          {allowedDirs.map((d) => (
            <div key={d} className="flex items-center justify-between bg-[#1a1a2e] rounded px-3 py-1.5 group">
              <span className="font-mono text-xs text-emerald-400">{d}</span>
              <button
                onClick={() => onRemoveDir(d)}
                className="text-gray-600 hover:text-red-400 text-xs opacity-0 group-hover:opacity-100 transition-opacity"
              >
                remove
              </button>
            </div>
          ))}
          {allowedDirs.length === 0 && (
            <div className="text-xs text-gray-600 py-2">No directory scope configured — all directories allowed.</div>
          )}
        </div>
      </div>

      {/* Deny File Patterns */}
      <div className="space-y-3">
        <div>
          <h4 className="text-xs font-medium text-gray-400 mb-1">Deny File Patterns</h4>
          <p className="text-xs text-gray-600 mb-2">
            Files matching these patterns will be blocked even within allowed directories.
          </p>
        </div>

        <div className="flex gap-2">
          <input
            value={patternInput}
            onChange={(e) => setPatternInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleAddPattern()}
            placeholder="e.g. *.env, **/secrets/**, .aws/*"
            className="flex-1 bg-[#1a1a2e] border border-gray-700 rounded px-3 py-1.5 text-xs text-gray-300 font-mono placeholder-gray-600 focus:outline-none focus:border-blue-600"
          />
          <button
            onClick={handleAddPattern}
            className="px-3 py-1.5 bg-blue-900/50 text-blue-300 rounded text-xs font-medium hover:bg-blue-900/70"
          >
            Add
          </button>
        </div>

        <div className="space-y-1">
          {patterns.map((p) => (
            <div key={p} className="flex items-center justify-between bg-[#1a1a2e] rounded px-3 py-1.5 group">
              <span className="font-mono text-xs text-gray-300">{p}</span>
              <button
                onClick={() => onRemove(p)}
                className="text-gray-600 hover:text-red-400 text-xs opacity-0 group-hover:opacity-100 transition-opacity"
              >
                remove
              </button>
            </div>
          ))}
          {patterns.length === 0 && (
            <div className="text-xs text-gray-600 py-2">No deny patterns configured.</div>
          )}
        </div>
      </div>
    </div>
  );
}