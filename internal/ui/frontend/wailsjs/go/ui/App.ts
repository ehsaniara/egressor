// Wails bindings — stubs that match the auto-generated API.

declare const window: any;

export function GetRecentSessions(limit: number): Promise<any[]> {
  return window.go.ui.App.GetRecentSessions(limit);
}

export function GetSession(id: string): Promise<any> {
  return window.go.ui.App.GetSession(id);
}

export function GetStats(): Promise<any> {
  return window.go.ui.App.GetStats();
}

export function GetDenyPatterns(): Promise<string[]> {
  return window.go.ui.App.GetDenyPatterns();
}

export function SetDenyPatterns(patterns: string[]): Promise<void> {
  return window.go.ui.App.SetDenyPatterns(patterns);
}

export function AddDenyPattern(pattern: string): Promise<void> {
  return window.go.ui.App.AddDenyPattern(pattern);
}

export function RemoveDenyPattern(pattern: string): Promise<void> {
  return window.go.ui.App.RemoveDenyPattern(pattern);
}

export function GetAllowedDirectories(): Promise<string[]> {
  return window.go.ui.App.GetAllowedDirectories();
}

export function SetAllowedDirectories(dirs: string[]): Promise<void> {
  return window.go.ui.App.SetAllowedDirectories(dirs);
}

export function AddAllowedDirectory(dir: string): Promise<void> {
  return window.go.ui.App.AddAllowedDirectory(dir);
}

export function RemoveAllowedDirectory(dir: string): Promise<void> {
  return window.go.ui.App.RemoveAllowedDirectory(dir);
}

export function IsPolicyBypassed(): Promise<boolean> {
  return window.go.ui.App.IsPolicyBypassed();
}

export function SetPolicyBypassed(bypassed: boolean): Promise<void> {
  return window.go.ui.App.SetPolicyBypassed(bypassed);
}

export function SaveConfig(): Promise<void> {
  return window.go.ui.App.SaveConfig();
}

export function StartProxy(): Promise<void> {
  return window.go.ui.App.StartProxy();
}

export function StopProxy(): Promise<void> {
  return window.go.ui.App.StopProxy();
}

export function IsProxyRunning(): Promise<boolean> {
  return window.go.ui.App.IsProxyRunning();
}

export function GetListenAddress(): Promise<string> {
  return window.go.ui.App.GetListenAddress();
}