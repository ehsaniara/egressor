import { useState, useEffect, useCallback } from 'react';
import { Session } from '../types';
import { GetRecentSessions, GetStats } from '../../wailsjs/go/ui/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import type { StoreStats } from '../types';

const MAX_SESSIONS = 500;

export function useSessions() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [stats, setStats] = useState<StoreStats>({ total_sessions: 0, blocked_count: 0, file_detections: 0 });

  useEffect(() => {
    GetRecentSessions(200).then((data) => {
      setSessions(data || []);
    });
    GetStats().then(setStats);
  }, []);

  useEffect(() => {
    const cancel = EventsOn('session:new', (session: Session) => {
      setSessions(prev => [session, ...prev].slice(0, MAX_SESSIONS));
      setStats(prev => ({
        total_sessions: prev.total_sessions + 1,
        blocked_count: prev.blocked_count + (session.exchanges?.some(e => e.blocked) ? 1 : 0),
        file_detections: prev.file_detections + (session.exchanges?.reduce((sum, e) => sum + (e.detected_files?.length || 0), 0) || 0),
      }));
    });
    return cancel;
  }, []);

  const refresh = useCallback(() => {
    GetRecentSessions(200).then((data) => setSessions(data || []));
    GetStats().then(setStats);
  }, []);

  return { sessions, stats, refresh };
}
