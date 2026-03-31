import type { Session } from '../types';

interface Props {
  sessions: Session[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}

function methodColor(method: string): string {
  switch (method) {
    case 'GET': return 'text-green-400';
    case 'POST': return 'text-blue-400';
    case 'PUT': return 'text-yellow-400';
    case 'DELETE': return 'text-red-400';
    case 'PATCH': return 'text-orange-400';
    default: return 'text-gray-400';
  }
}

function statusColor(code: number): string {
  if (code >= 200 && code < 300) return 'text-green-400';
  if (code >= 300 && code < 400) return 'text-yellow-400';
  if (code >= 400 && code < 500) return 'text-red-400';
  if (code >= 500) return 'text-red-500';
  return 'text-gray-400';
}

export function SessionTable({ sessions, selectedId, onSelect }: Props) {
  return (
    <div className="flex-1 overflow-auto">
      <table className="w-full text-xs">
        <thead className="sticky top-0 bg-[#12121f] z-10">
          <tr className="text-gray-500 border-b border-gray-800">
            <th className="text-left py-2 px-3 font-medium w-20">Method</th>
            <th className="text-left py-2 px-3 font-medium">Host</th>
            <th className="text-left py-2 px-3 font-medium">Path</th>
            <th className="text-left py-2 px-3 font-medium w-16">Status</th>
            <th className="text-left py-2 px-3 font-medium w-16">Files</th>
            <th className="text-left py-2 px-3 font-medium w-20">Duration</th>
          </tr>
        </thead>
        <tbody>
          {sessions.map((sess) => {
            const exchange = sess.exchanges?.[0];
            const isBlocked = exchange?.blocked;
            const isSelected = sess.session_id === selectedId;
            const fileCount = sess.exchanges?.reduce((sum, e) => sum + (e.detected_files?.length || 0), 0) || 0;
            const method = exchange?.method || '-';
            const urlPath = exchange?.url ? new URL(exchange.url).pathname : '-';
            const status = exchange?.status_code || 0;

            return (
              <tr
                key={sess.session_id}
                onClick={() => onSelect(sess.session_id)}
                className={`cursor-pointer border-b border-gray-800/50 transition-colors ${
                  isSelected
                    ? 'bg-blue-900/30'
                    : isBlocked
                      ? 'bg-red-900/20 hover:bg-red-900/30'
                      : 'hover:bg-gray-800/30'
                }`}
              >
                <td className={`py-1.5 px-3 font-mono font-medium ${methodColor(method)}`}>
                  {method}
                </td>
                <td className="py-1.5 px-3 text-gray-300 truncate max-w-48">
                  {sess.target_host}
                </td>
                <td className="py-1.5 px-3 text-gray-400 truncate max-w-64 font-mono">
                  {urlPath}
                </td>
                <td className={`py-1.5 px-3 font-mono ${isBlocked ? 'text-red-400' : statusColor(status)}`}>
                  {isBlocked ? 'BLK' : status || '-'}
                </td>
                <td className="py-1.5 px-3">
                  {fileCount > 0 && (
                    <span className="bg-blue-900/50 text-blue-300 px-1.5 py-0.5 rounded text-[10px]">
                      {fileCount}
                    </span>
                  )}
                </td>
                <td className="py-1.5 px-3 text-gray-500 font-mono">
                  {sess.duration_ms ? `${sess.duration_ms}ms` : '-'}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
      {sessions.length === 0 && (
        <div className="text-center text-gray-600 py-16">
          No sessions captured yet. Configure your tools to use the proxy.
        </div>
      )}
    </div>
  );
}
