/**
 * Type definitions for the Forge Agent WASM bridge interface
 * 
 * This file defines the types used for communication between
 * the Go host and the WASM agent.
 */

/**
 * Configuration for initializing the agent
 */
export interface AgentConfig {
  /** LLM provider to use (claude, openai, gemini) */
  provider: string;
  /** Unique session identifier */
  sessionId?: string;
  /** Additional provider-specific options */
  options?: Record<string, unknown>;
}

/**
 * Message in a conversation
 */
export interface Message {
  /** Role of the message sender */
  role: 'user' | 'assistant' | 'system';
  /** Message content */
  content: string;
}

/**
 * Response from the agent
 */
export interface AgentResponse {
  /** Response content from the LLM */
  content?: string;
  /** Tool calls requested by the agent */
  toolCalls?: ToolCall[];
  /** Error message if something went wrong */
  error?: string;
}

/**
 * A tool call requested by the agent
 */
export interface ToolCall {
  /** Tool identifier */
  tool: string;
  /** Tool parameters */
  params: Record<string, unknown>;
}

/**
 * Result of a tool execution
 */
export interface ToolResult {
  /** Whether the tool executed successfully */
  success: boolean;
  /** Tool output or error message */
  output?: string;
  /** Error message if execution failed */
  error?: string;
}

/**
 * File information returned by listDirectory
 */
export interface FileInfo {
  /** File name */
  name: string;
  /** Full path to file */
  path: string;
  /** Whether this is a directory */
  isDir: boolean;
  /** File size in bytes */
  size: number;
}

/**
 * Code search result
 */
export interface SearchResult {
  /** File path where match was found */
  filePath: string;
  /** Line number of match */
  line: number;
  /** Content of the line */
  content: string;
  /** The matched text */
  match: string;
}

/**
 * Agent state snapshot
 */
export interface AgentState {
  /** Whether the agent is initialized */
  initialized: boolean;
  /** Current session ID */
  sessionId: string;
  /** Active provider */
  provider: string;
  /** Number of messages in history */
  messageCount: number;
}

/**
 * Host function interface - these are provided by the Go runtime
 */
export interface HostFunctions {
  /** Read a file from disk */
  hostReadFile(path: string): Uint8Array;
  
  /** Write a file to disk */
  hostWriteFile(path: string, content: Uint8Array): void;
  
  /** Execute a shell command */
  hostExecuteCommand(cmd: string, args: string[]): string;
  
  /** List directory contents */
  hostListDirectory(path: string): string;
  
  /** Search code in the project */
  hostSearchCode(query: string): string;
  
  /** Send request to LLM */
  hostSendLLMRequest(provider: string, messages: string): string;
  
  /** Log a message to the host */
  hostLog(message: string): void;
}

/**
 * Exported WASM functions - these are called from Go
 */
export interface WASMExports {
  /** Initialize the agent */
  initialize(config: string): i32;
  
  /** Process a user message */
  processMessage(message: string): string;
  
  /** Execute a tool call */
  executeTool(toolCall: string): string;
  
  /** Get current state */
  getState(): string;
  
  /** Reset agent state */
  reset(): void;
}
