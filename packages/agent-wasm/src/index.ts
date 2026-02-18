/**
 * Forge Agent WASM Entry Point
 * 
 * This is the main entry point for the OpenCode agent when running as WASM.
 * It exports functions that can be called from the Go runtime.
 */

// Declare host functions provided by Go runtime
declare function hostReadFile(path: string): Uint8Array;
declare function hostWriteFile(path: string, content: Uint8Array): void;
declare function hostExecuteCommand(cmd: string, args: string[]): string;
declare function hostListDirectory(path: string): string;
declare function hostSearchCode(query: string): string;
declare function hostSendLLMRequest(provider: string, messages: string): string;
declare function hostLog(message: string): void;

// Agent state
let agentState: AgentState = {
  initialized: false,
  sessionId: '',
  provider: 'claude',
  messageHistory: []
};

interface AgentState {
  initialized: boolean;
  sessionId: string;
  provider: string;
  messageHistory: Message[];
}

interface Message {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

/**
 * Initialize the agent with configuration
 * @param config JSON string with agent configuration
 * @returns 0 on success, non-zero on error
 */
export function initialize(config: string): i32 {
  try {
    hostLog('Initializing Forge agent...');
    
    // Parse configuration
    const configObj = JSON.parse(config);
    agentState.provider = configObj.provider || 'claude';
    agentState.sessionId = configObj.sessionId || generateSessionId();
    agentState.initialized = true;
    
    hostLog(`Agent initialized with provider: ${agentState.provider}`);
    return 0;
  } catch (e) {
    hostLog(`Initialization failed: ${e.message}`);
    return 1;
  }
}

/**
 * Process a user message and return agent response
 * @param message User message
 * @returns Response from agent (JSON string)
 */
export function processMessage(message: string): string {
  if (!agentState.initialized) {
    return JSON.stringify({ error: 'Agent not initialized' });
  }

  try {
    // Add user message to history
    agentState.messageHistory.push({
      role: 'user',
      content: message
    });

    // Prepare messages for LLM
    const messages = JSON.stringify(agentState.messageHistory);
    
    // Send to LLM via host function
    const response = hostSendLLMRequest(agentState.provider, messages);
    
    // Parse and store response
    const responseObj = JSON.parse(response);
    
    if (responseObj.content) {
      agentState.messageHistory.push({
        role: 'assistant',
        content: responseObj.content
      });
    }

    return response;
  } catch (e) {
    return JSON.stringify({ error: e.message });
  }
}

/**
 * Execute a tool call requested by the agent
 * @param toolCall JSON string describing the tool call
 * @returns Tool execution result (JSON string)
 */
export function executeTool(toolCall: string): string {
  try {
    const call = JSON.parse(toolCall);
    const { tool, params } = call;

    switch (tool) {
      case 'readFile': {
        const content = hostReadFile(params.path);
        return JSON.stringify({ content: content.toString() });
      }

      case 'writeFile': {
        const data = Uint8Array.wrap(String.UTF8.encode(params.content));
        hostWriteFile(params.path, data);
        return JSON.stringify({ success: true });
      }

      case 'executeCommand': {
        const output = hostExecuteCommand(params.command, params.args || []);
        return JSON.stringify({ output });
      }

      case 'listDirectory': {
        const listing = hostListDirectory(params.path);
        return JSON.stringify({ listing: JSON.parse(listing) });
      }

      case 'searchCode': {
        const results = hostSearchCode(params.query);
        return JSON.stringify({ results: JSON.parse(results) });
      }

      default:
        return JSON.stringify({ error: `Unknown tool: ${tool}` });
    }
  } catch (e) {
    return JSON.stringify({ error: e.message });
  }
}

/**
 * Get current agent state
 * @returns JSON string with agent state
 */
export function getState(): string {
  return JSON.stringify({
    initialized: agentState.initialized,
    sessionId: agentState.sessionId,
    provider: agentState.provider,
    messageCount: agentState.messageHistory.length
  });
}

/**
 * Reset agent state
 */
export function reset(): void {
  agentState = {
    initialized: false,
    sessionId: '',
    provider: 'claude',
    messageHistory: []
  };
  hostLog('Agent state reset');
}

/**
 * Generate a unique session ID
 */
function generateSessionId(): string {
  const timestamp = Date.now().toString(36);
  const random = Math.random().toString(36).substr(2, 9);
  return `${timestamp}-${random}`;
}
