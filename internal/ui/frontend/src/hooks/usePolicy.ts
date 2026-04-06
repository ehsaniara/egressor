import { useState, useEffect, useCallback } from 'react';
import {
  GetDenyPatterns,
  SetDenyPatterns,
  AddDenyPattern,
  RemoveDenyPattern,
  GetAllowedDirectories,
  AddAllowedDirectory,
  RemoveAllowedDirectory,
  GetDenyContentKeywords,
  AddDenyContentKeyword,
  RemoveDenyContentKeyword,
  GetContentKeywordWhitelist,
  RemoveFromContentKeywordWhitelist,
  GetContentKeywordBlacklist,
  RemoveFromContentKeywordBlacklist,
  IsPolicyBypassed,
  SetPolicyBypassed,
  SaveConfig,
} from '../../wailsjs/go/ui/App';

export function usePolicy() {
  const [patterns, setPatterns] = useState<string[]>([]);
  const [allowedDirs, setAllowedDirs] = useState<string[]>([]);
  const [contentKeywords, setContentKeywords] = useState<string[]>([]);
  const [whitelist, setWhitelist] = useState<string[]>([]);
  const [blacklist, setBlacklist] = useState<string[]>([]);
  const [bypassed, setBypassed] = useState(false);

  useEffect(() => {
    GetDenyPatterns().then(setPatterns);
    GetAllowedDirectories().then(setAllowedDirs);
    GetDenyContentKeywords().then(setContentKeywords);
    GetContentKeywordWhitelist().then(setWhitelist);
    GetContentKeywordBlacklist().then(setBlacklist);
    IsPolicyBypassed().then(setBypassed);
  }, []);

  // Deny patterns
  const addPattern = useCallback(async (pattern: string) => {
    await AddDenyPattern(pattern);
    setPatterns(await GetDenyPatterns());
  }, []);

  const removePattern = useCallback(async (pattern: string) => {
    await RemoveDenyPattern(pattern);
    setPatterns(await GetDenyPatterns());
  }, []);

  const updatePatterns = useCallback(async (newPatterns: string[]) => {
    await SetDenyPatterns(newPatterns);
    setPatterns(newPatterns);
  }, []);

  // Allowed directories
  const addDirectory = useCallback(async (dir: string) => {
    await AddAllowedDirectory(dir);
    setAllowedDirs(await GetAllowedDirectories());
  }, []);

  const removeDirectory = useCallback(async (dir: string) => {
    await RemoveAllowedDirectory(dir);
    setAllowedDirs(await GetAllowedDirectories());
  }, []);

  // Content keywords
  const addContentKeyword = useCallback(async (keyword: string) => {
    await AddDenyContentKeyword(keyword);
    setContentKeywords(await GetDenyContentKeywords());
  }, []);

  const removeContentKeyword = useCallback(async (keyword: string) => {
    await RemoveDenyContentKeyword(keyword);
    setContentKeywords(await GetDenyContentKeywords());
  }, []);

  // Whitelist
  const removeFromWhitelist = useCallback(async (path: string) => {
    await RemoveFromContentKeywordWhitelist(path);
    setWhitelist(await GetContentKeywordWhitelist());
  }, []);

  // Blacklist
  const removeFromBlacklist = useCallback(async (path: string) => {
    await RemoveFromContentKeywordBlacklist(path);
    setBlacklist(await GetContentKeywordBlacklist());
  }, []);

  // Bypass
  const toggleBypassed = useCallback(async () => {
    const next = !bypassed;
    await SetPolicyBypassed(next);
    setBypassed(next);
  }, [bypassed]);

  const save = useCallback(async () => {
    await SaveConfig();
  }, []);

  return {
    patterns, bypassed, addPattern, removePattern, updatePatterns,
    allowedDirs, addDirectory, removeDirectory,
    contentKeywords, addContentKeyword, removeContentKeyword,
    whitelist, removeFromWhitelist,
    blacklist, removeFromBlacklist,
    toggleBypassed, save,
  };
}