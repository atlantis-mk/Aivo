export function EventsOn(eventName: string, callback: (...data: any[]) => void): () => void;
export function EventsOnMultiple(eventName: string, callback: (...data: any[]) => void, maxCallbacks: number): () => void;
export function EventsOnce(eventName: string, callback: (...data: any[]) => void): () => void;
export function EventsOff(eventName: string, callback?: (...data: any[]) => void): void;
export function EventsOffAll(): void;
export function EventsEmit(...args: unknown[]): void;

export function BrowserOpenURL(url: string): Promise<void> | void;
export function WindowToggleMaximise(): Promise<void> | void;
export function WindowMaximise(): Promise<void> | void;
export function WindowReload(): void;
export function WindowReloadApp(): void;
export function WindowSetTitle(title: string): void;
export function WindowIsFullscreen(): Promise<boolean>;
export function WindowGetSize(): Promise<{ w: number; h: number }>;
export function WindowGetPosition(): Promise<{ x: number; y: number }>;
export function WindowIsMaximised(): Promise<boolean>;
export function Environment(): Promise<{ platform?: string }>;
export function ClipboardGetText(): Promise<string>;
export function ClipboardSetText(text: string): Promise<void> | undefined;

export function LogPrint(message: string): void;
export function LogTrace(message: string): void;
export function LogDebug(message: string): void;
export function LogInfo(message: string): void;
export function LogWarning(message: string): void;
export function LogError(message: string): void;
export function LogFatal(message: string): void;
