import { useState } from 'react';
import { useSessions } from './hooks/useSessions';
import { usePolicy } from './hooks/usePolicy';
import { SessionTable } from './components/SessionTable';
import { SessionDetail } from './components/SessionDetail';
import { ProxyControls } from './components/ProxyControls';
import { PolicyEditor } from './components/PolicyEditor';

type Tab = 'sessions' | 'policy';

function App() {
  const { sessions, stats } = useSessions();
  const policy = usePolicy();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>('sessions');

  const selectedSession = selectedId ? sessions.find(s => s.session_id === selectedId) : null;

  return (
    <div className="flex flex-col h-screen bg-[#0f0f1a] text-gray-200">
      {/* Top bar */}
      <div className="flex items-center px-4 py-2 bg-[#0d0d1a] border-b border-gray-800">
        <h1 className="text-sm font-bold text-gray-200 tracking-wider mr-6">EGRESSOR</h1>
        <div className="flex gap-1">
          <button
            onClick={() => setActiveTab('sessions')}
            className={`px-3 py-1 rounded text-xs font-medium ${
              activeTab === 'sessions' ? 'bg-gray-700/50 text-gray-200' : 'text-gray-500 hover:text-gray-300'
            }`}
          >
            Sessions
          </button>
          <button
            onClick={() => setActiveTab('policy')}
            className={`px-3 py-1 rounded text-xs font-medium ${
              activeTab === 'policy' ? 'bg-gray-700/50 text-gray-200' : 'text-gray-500 hover:text-gray-300'
            }`}
          >
            Policy
          </button>
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 flex overflow-hidden">
        {activeTab === 'sessions' ? (
          <>
            {/* Session list */}
            <div className={`flex flex-col ${selectedSession ? 'w-[45%]' : 'w-full'} border-r border-gray-800`}>
              <SessionTable
                sessions={sessions}
                selectedId={selectedId}
                onSelect={setSelectedId}
              />
            </div>

            {/* Detail panel */}
            {selectedSession && (
              <div className="flex-1 flex flex-col">
                <div className="flex justify-end px-2 py-1 bg-[#0d0d1a] border-b border-gray-800">
                  <button
                    onClick={() => setSelectedId(null)}
                    className="text-gray-600 hover:text-gray-300 text-xs px-2"
                  >
                    Close
                  </button>
                </div>
                <div className="flex-1 overflow-auto">
                  <SessionDetail session={selectedSession} />
                </div>
              </div>
            )}
          </>
        ) : (
          <div className="w-full max-w-2xl">
            <PolicyEditor
              patterns={policy.patterns}
              onAdd={policy.addPattern}
              onRemove={policy.removePattern}
              onSave={policy.save}
            />
          </div>
        )}
      </div>

      {/* Bottom bar */}
      <ProxyControls
        stats={stats}
        bypassed={policy.bypassed}
        onToggleBypassed={policy.toggleBypassed}
      />
    </div>
  );
}

export default App;
