import type { InterceptedExchange } from '../types';
import { JsonViewer } from './JsonViewer';

interface Props {
  exchange: InterceptedExchange;
}

export function RequestPane({ exchange }: Props) {
  return (
    <div className="space-y-3">
      <div>
        <h4 className="text-xs font-medium text-gray-500 mb-1 uppercase tracking-wider">Request</h4>
        <div className="flex items-center gap-2 text-sm">
          <span className="font-mono font-bold text-blue-400">{exchange.method}</span>
          <span className="font-mono text-gray-300 truncate">{exchange.url}</span>
        </div>
      </div>

      {exchange.detected_files?.length > 0 && (
        <div>
          <h4 className="text-xs font-medium text-gray-500 mb-1 uppercase tracking-wider">Detected Files</h4>
          <div className="space-y-1">
            {exchange.detected_files.map((f, i) => (
              <div key={i} className="flex items-center gap-2 text-xs">
                <span className="font-mono text-yellow-300">{f.path}</span>
                <span className="text-gray-600">({f.source})</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {exchange.blocked && (
        <div className="bg-red-900/30 border border-red-800 rounded p-2 text-xs text-red-300">
          Blocked: {exchange.block_reason}
        </div>
      )}

      {exchange.request_headers && Object.keys(exchange.request_headers).length > 0 && (
        <div>
          <h4 className="text-xs font-medium text-gray-500 mb-1 uppercase tracking-wider">Headers</h4>
          <div className="bg-[#1a1a2e] rounded p-2 text-xs space-y-0.5">
            {Object.entries(exchange.request_headers).map(([k, v]) => (
              <div key={k}>
                <span className="text-purple-400">{k}</span>
                <span className="text-gray-600">: </span>
                <span className="text-gray-300">{v}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {exchange.request_body && (
        <div>
          <h4 className="text-xs font-medium text-gray-500 mb-1 uppercase tracking-wider">Body</h4>
          <JsonViewer data={exchange.request_body} />
        </div>
      )}
    </div>
  );
}
