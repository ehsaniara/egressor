import { useState, useEffect } from 'react';
import { IsProxyRunning, StartProxy, StopProxy, GetListenAddress } from '../../wailsjs/go/ui/App';
import type { StoreStats } from '../types';

interface Props {
  stats: StoreStats;
  bypassed: boolean;
  onToggleBypassed: () => void;
}

export function ProxyControls({ stats, bypassed, onToggleBypassed }: Props) {
  const [running, setRunning] = useState(false);
  const [address, setAddress] = useState('');

  useEffect(() => {
    IsProxyRunning().then(setRunning);
    GetListenAddress().then(setAddress);
  }, []);

  const toggle = async () => {
    if (running) {
      await StopProxy();
    } else {
      await StartProxy();
    }
    setRunning(await IsProxyRunning());
  };

  return (
    <div className="flex items-center gap-4 px-4 py-2 bg-[#0d0d1a] border-t border-gray-800 text-sm">
      <div className="flex items-center gap-2">
        <div className={`w-2 h-2 rounded-full ${running ? 'bg-green-400' : 'bg-red-400'}`} />
        <span className="text-gray-400">{running ? 'Running' : 'Stopped'}</span>
      </div>

      <span className="text-gray-500 font-mono text-xs">{address}</span>

      <button
        onClick={toggle}
        className={`px-3 py-1 rounded text-xs font-medium ${
          running
            ? 'bg-red-900/50 text-red-300 hover:bg-red-900/70'
            : 'bg-green-900/50 text-green-300 hover:bg-green-900/70'
        }`}
      >
        {running ? 'Stop' : 'Start'}
      </button>

      <button
        onClick={onToggleBypassed}
        className={`px-3 py-1 rounded text-xs font-medium ${
          bypassed
            ? 'bg-yellow-900/50 text-yellow-300 hover:bg-yellow-900/70'
            : 'bg-gray-700/50 text-gray-300 hover:bg-gray-700/70'
        }`}
      >
        {bypassed ? 'Resume Policy' : 'Pause Policy'}
      </button>

      <div className="flex-1" />

      <div className="flex gap-4 text-xs text-gray-500">
        <span>Sessions: <span className="text-gray-300">{stats.total_sessions}</span></span>
        <span>Blocked: <span className="text-red-400">{stats.blocked_count}</span></span>
        <span>Files: <span className="text-blue-400">{stats.file_detections}</span></span>
      </div>
    </div>
  );
}
