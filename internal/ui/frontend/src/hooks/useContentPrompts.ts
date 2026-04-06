import { useState, useEffect, useCallback } from 'react';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import {
  ResolveContentPrompt,
  ResolveContentPromptForFile,
} from '../../wailsjs/go/ui/App';
import type { ContentPrompt, PromptAction } from '../types';

export function useContentPrompts() {
  const [queue, setQueue] = useState<ContentPrompt[]>([]);

  useEffect(() => {
    const cancel = EventsOn('content:prompt', (prompt: ContentPrompt) => {
      setQueue(prev => [...prev, prompt]);
    });
    return () => { cancel(); };
  }, []);

  const resolve = useCallback(async (promptId: string, action: PromptAction, filePaths: string[]) => {
    // Persist whitelist/blacklist for each file if needed
    if (action === 'allow_always' || action === 'block_always') {
      for (const fp of filePaths) {
        await ResolveContentPromptForFile(action, fp);
      }
    }
    // Unblock the interceptor
    await ResolveContentPrompt(promptId, action);
    setQueue(prev => prev.filter(p => p.id !== promptId));
  }, []);

  return { current: queue[0] ?? null, queueLength: queue.length, resolve };
}