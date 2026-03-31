import { useState, useEffect, useCallback } from 'react';
import {
  GetDenyPatterns,
  SetDenyPatterns,
  AddDenyPattern,
  RemoveDenyPattern,
  IsPolicyBypassed,
  SetPolicyBypassed,
  SaveConfig,
} from '../../wailsjs/go/ui/App';

export function usePolicy() {
  const [patterns, setPatterns] = useState<string[]>([]);
  const [bypassed, setBypassed] = useState(false);

  useEffect(() => {
    GetDenyPatterns().then(setPatterns);
    IsPolicyBypassed().then(setBypassed);
  }, []);

  const addPattern = useCallback(async (pattern: string) => {
    await AddDenyPattern(pattern);
    const updated = await GetDenyPatterns();
    setPatterns(updated);
  }, []);

  const removePattern = useCallback(async (pattern: string) => {
    await RemoveDenyPattern(pattern);
    const updated = await GetDenyPatterns();
    setPatterns(updated);
  }, []);

  const updatePatterns = useCallback(async (newPatterns: string[]) => {
    await SetDenyPatterns(newPatterns);
    setPatterns(newPatterns);
  }, []);

  const toggleBypassed = useCallback(async () => {
    const next = !bypassed;
    await SetPolicyBypassed(next);
    setBypassed(next);
  }, [bypassed]);

  const save = useCallback(async () => {
    await SaveConfig();
  }, []);

  return { patterns, bypassed, addPattern, removePattern, updatePatterns, toggleBypassed, save };
}
