declare const window: any;

export function EventsOn(eventName: string, callback: (...data: any[]) => void): () => void {
  return window.runtime.EventsOn(eventName, callback);
}

export function EventsOff(eventName: string): void {
  window.runtime.EventsOff(eventName);
}