import { useState } from 'react';

interface Props {
  patterns: string[];
  onAdd: (pattern: string) => void;
  onRemove: (pattern: string) => void;
  allowedDirs: string[];
  onAddDir: (dir: string) => void;
  onRemoveDir: (dir: string) => void;
  contentTags: string[];
  onAddTag: (tag: string) => void;
  onRemoveTag: (tag: string) => void;
  contentKeywords: string[];
  onAddKeyword: (keyword: string) => void;
  onRemoveKeyword: (keyword: string) => void;
  whitelist: string[];
  onRemoveWhitelist: (path: string) => void;
  blacklist: string[];
  onRemoveBlacklist: (path: string) => void;
  onSave: () => void;
}

export function PolicyEditor({
  patterns, onAdd, onRemove,
  allowedDirs, onAddDir, onRemoveDir,
  contentTags, onAddTag, onRemoveTag,
  contentKeywords, onAddKeyword, onRemoveKeyword,
  whitelist, onRemoveWhitelist,
  blacklist, onRemoveBlacklist,
  onSave,
}: Props) {
  const [patternInput, setPatternInput] = useState('');
  const [dirInput, setDirInput] = useState('');
  const [tagInput, setTagInput] = useState('');
  const [keywordInput, setKeywordInput] = useState('');
  const [saved, setSaved] = useState(false);

  const handleAddPattern = () => {
    const trimmed = patternInput.trim();
    if (trimmed && !patterns.includes(trimmed)) {
      onAdd(trimmed);
      setPatternInput('');
    }
  };

  const handleAddTag = () => {
    const trimmed = tagInput.trim();
    if (trimmed && !contentTags.includes(trimmed)) {
      onAddTag(trimmed);
      setTagInput('');
    }
  };

  const handleAddDir = () => {
    const trimmed = dirInput.trim();
    if (trimmed && !allowedDirs.includes(trimmed)) {
      onAddDir(trimmed);
      setDirInput('');
    }
  };

  const handleAddKeyword = () => {
    const trimmed = keywordInput.trim();
    if (trimmed && !contentKeywords.includes(trimmed)) {
      onAddKeyword(trimmed);
      setKeywordInput('');
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
      <Section title="Allowed Directories" description="Files outside these directories will be blocked. Leave empty to allow all.">
        <AddInput
          value={dirInput}
          onChange={setDirInput}
          onAdd={handleAddDir}
          placeholder="e.g. ~/Projects/my-app"
        />
        <ItemList items={allowedDirs} onRemove={onRemoveDir} color="text-emerald-400" empty="No directory scope configured — all directories allowed." />
      </Section>

      {/* Deny File Patterns */}
      <Section title="Deny File Patterns" description="Files matching these patterns will be blocked even within allowed directories.">
        <AddInput
          value={patternInput}
          onChange={setPatternInput}
          onAdd={handleAddPattern}
          placeholder="e.g. *.env, **/secrets/**, .aws/*"
        />
        <ItemList items={patterns} onRemove={onRemove} color="text-gray-300" empty="No deny patterns configured." />
      </Section>

      {/* Content Tags (hard block) */}
      <Section title="Content Tags (hard block)" description="Developers add these tags (e.g. // NO_LLM) to files that must never be sent to LLMs. Requests containing these tags are blocked immediately — no prompt.">
        <AddInput
          value={tagInput}
          onChange={setTagInput}
          onAdd={handleAddTag}
          placeholder='e.g. NO_LLM'
        />
        <ItemList items={contentTags} onRemove={onRemoveTag} color="text-red-300" empty="No content tags configured." />
      </Section>

      {/* Content Keywords (interactive) */}
      <Section title="Content Keywords (interactive)" description="Request bodies containing these keywords will prompt for approval. Users can whitelist or blacklist files to avoid repeated prompts.">
        <AddInput
          value={keywordInput}
          onChange={setKeywordInput}
          onAdd={handleAddKeyword}
          placeholder='e.g. CONFIDENTIAL, INTERNAL ONLY'
        />
        <ItemList items={contentKeywords} onRemove={onRemoveKeyword} color="text-yellow-300" empty="No content keywords configured." />
      </Section>

      {/* Whitelist */}
      {whitelist.length > 0 && (
        <Section title="Keyword Whitelist" description="Files approved via 'Allow Always' — keyword checks are skipped for these.">
          <ItemList items={whitelist} onRemove={onRemoveWhitelist} color="text-green-400" empty="" />
        </Section>
      )}

      {/* Blacklist */}
      {blacklist.length > 0 && (
        <Section title="Keyword Blacklist" description="Files blocked via 'Block Always' — automatically blocked when keywords match.">
          <ItemList items={blacklist} onRemove={onRemoveBlacklist} color="text-red-400" empty="" />
        </Section>
      )}
    </div>
  );
}

function Section({ title, description, children }: { title: string; description: string; children: React.ReactNode }) {
  return (
    <div className="space-y-3">
      <div>
        <h4 className="text-xs font-medium text-gray-400 mb-1">{title}</h4>
        <p className="text-xs text-gray-600 mb-2">{description}</p>
      </div>
      {children}
    </div>
  );
}

function AddInput({ value, onChange, onAdd, placeholder }: {
  value: string;
  onChange: (v: string) => void;
  onAdd: () => void;
  placeholder: string;
}) {
  return (
    <div className="flex gap-2">
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && onAdd()}
        placeholder={placeholder}
        className="flex-1 bg-[#1a1a2e] border border-gray-700 rounded px-3 py-1.5 text-xs text-gray-300 font-mono placeholder-gray-600 focus:outline-none focus:border-blue-600"
      />
      <button
        onClick={onAdd}
        className="px-3 py-1.5 bg-blue-900/50 text-blue-300 rounded text-xs font-medium hover:bg-blue-900/70"
      >
        Add
      </button>
    </div>
  );
}

function ItemList({ items, onRemove, color, empty }: {
  items: string[];
  onRemove: (item: string) => void;
  color: string;
  empty: string;
}) {
  return (
    <div className="space-y-1">
      {items.map((item) => (
        <div key={item} className="flex items-center justify-between bg-[#1a1a2e] rounded px-3 py-1.5 group">
          <span className={`font-mono text-xs ${color}`}>{item}</span>
          <button
            onClick={() => onRemove(item)}
            className="text-gray-600 hover:text-red-400 text-xs opacity-0 group-hover:opacity-100 transition-opacity"
          >
            remove
          </button>
        </div>
      ))}
      {items.length === 0 && empty && (
        <div className="text-xs text-gray-600 py-2">{empty}</div>
      )}
    </div>
  );
}