import { useState } from 'react';
import type { Session } from '../types';
import { RequestPane } from './RequestPane';
import { ResponsePane } from './ResponsePane';

interface Props {
  session: Session;
}

export function SessionDetail({ session }: Props) {
  const [activeTab, setActiveTab] = useState(0);
  const exchanges = session.exchanges || [];

  if (exchanges.length === 0) {
    return (
      <div className="text-center text-gray-600 py-8 text-sm">
        No exchanges captured for this session.
        {session.error && <div className="text-red-400 mt-2">Error: {session.error}</div>}
      </div>
    );
  }

  const exchange = exchanges[activeTab];

  return (
    <div className="flex flex-col h-full">
      {/* Session header */}
      <div className="px-4 py-2 border-b border-gray-800 bg-[#0d0d1a]">
        <div className="flex items-center gap-2 text-sm">
          <span className="text-gray-400">Session</span>
          <span className="font-mono text-gray-300">{session.session_id}</span>
          <span className="text-gray-600">|</span>
          <span className="text-gray-400">{session.target_host}:{session.target_port}</span>
          {session.duration_ms > 0 && (
            <>
              <span className="text-gray-600">|</span>
              <span className="text-gray-500">{session.duration_ms}ms</span>
            </>
          )}
        </div>
      </div>

      {/* Exchange tabs */}
      {exchanges.length > 1 && (
        <div className="flex gap-1 px-4 py-1 border-b border-gray-800 bg-[#0d0d1a]">
          {exchanges.map((ex, i) => (
            <button
              key={i}
              onClick={() => setActiveTab(i)}
              className={`px-2 py-1 rounded text-xs font-mono ${
                i === activeTab
                  ? 'bg-blue-900/50 text-blue-300'
                  : 'text-gray-500 hover:text-gray-300'
              }`}
            >
              {ex.method} {ex.blocked && '(blocked)'}
            </button>
          ))}
        </div>
      )}

      {/* Request/Response split */}
      <div className="flex-1 overflow-auto grid grid-cols-2 divide-x divide-gray-800">
        <div className="p-4 overflow-auto">
          <RequestPane exchange={exchange} />
        </div>
        <div className="p-4 overflow-auto">
          <ResponsePane exchange={exchange} />
        </div>
      </div>
    </div>
  );
}
