import type { InterceptedExchange } from '../types';
import { JsonViewer } from './JsonViewer';

interface Props {
  exchange: InterceptedExchange;
}

export function ResponsePane({ exchange }: Props) {
  if (exchange.blocked) {
    return (
      <div className="text-center text-red-400 py-8 text-sm">
        Request was blocked — no response.
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div>
        <h4 className="text-xs font-medium text-gray-500 mb-1 uppercase tracking-wider">Response</h4>
        <span className={`font-mono text-sm font-bold ${
          exchange.status_code >= 200 && exchange.status_code < 300 ? 'text-green-400' :
          exchange.status_code >= 400 ? 'text-red-400' : 'text-yellow-400'
        }`}>
          {exchange.status_code}
        </span>
      </div>

      {exchange.response_headers && Object.keys(exchange.response_headers).length > 0 && (
        <div>
          <h4 className="text-xs font-medium text-gray-500 mb-1 uppercase tracking-wider">Headers</h4>
          <div className="bg-[#1a1a2e] rounded p-2 text-xs space-y-0.5">
            {Object.entries(exchange.response_headers).map(([k, v]) => (
              <div key={k}>
                <span className="text-purple-400">{k}</span>
                <span className="text-gray-600">: </span>
                <span className="text-gray-300">{v}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {exchange.response_body && (
        <div>
          <h4 className="text-xs font-medium text-gray-500 mb-1 uppercase tracking-wider">Body</h4>
          <JsonViewer data={exchange.response_body} />
        </div>
      )}
    </div>
  );
}
