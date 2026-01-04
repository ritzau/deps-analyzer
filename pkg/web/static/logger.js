/**
 * Structured logging for the frontend
 *
 * Log Levels:
 * - TRACE: Very spammy, debug-time only
 * - DEBUG: Internal component behavior
 * - INFO: User-facing operations
 * - WARN: Should be monitored
 * - ERROR: Logical bugs that shouldn't happen
 */

const LogLevel = {
  TRACE: 0,
  DEBUG: 1,
  INFO: 2,
  WARN: 3,
  ERROR: 4,
};

class Logger {
  constructor() {
    this.level = LogLevel.INFO; // Default level
    this.requestIDCounter = 0;
    this.sendToBackend = true; // Whether to send logs to backend (enabled by default)
    this.logBuffer = []; // Buffer for batching logs
    this.batchSize = 10; // Send after this many logs
    this.batchTimeout = 5000; // Send after this many milliseconds
    this.batchTimer = null; // Timer for batch sending
  }

  /**
   * Set the minimum log level
   * @param {number} level - LogLevel enum value
   */
  setLevel(level) {
    this.level = level;
  }

  /**
   * Enable sending logs to backend
   * @param {boolean} enabled - Whether to send logs to backend
   */
  enableBackendLogging(enabled = true) {
    this.sendToBackend = enabled;
    if (!enabled && this.batchTimer) {
      clearTimeout(this.batchTimer);
      this.batchTimer = null;
    }
  }

  /**
   * Flush buffered logs to backend immediately
   * @private
   */
  async _flushLogs() {
    if (this.logBuffer.length === 0) return;

    const logsToSend = this.logBuffer.splice(0, this.logBuffer.length);

    if (this.batchTimer) {
      clearTimeout(this.batchTimer);
      this.batchTimer = null;
    }

    try {
      await fetch("/api/logs", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ logs: logsToSend }),
      });
    } catch (error) {
      // Don't log errors sending logs to avoid infinite loop
      console.error("[Logger] Failed to send logs to backend:", error);
    }
  }

  /**
   * Add log to buffer and maybe flush
   * @private
   */
  _bufferLog(logData) {
    if (!this.sendToBackend) return;

    this.logBuffer.push(logData);

    // Flush if buffer is full
    if (this.logBuffer.length >= this.batchSize) {
      this._flushLogs();
      return;
    }

    // Set timer to flush after timeout
    if (!this.batchTimer) {
      this.batchTimer = setTimeout(() => {
        this._flushLogs();
      }, this.batchTimeout);
    }
  }

  /**
   * Generate a new request ID for tracking
   * @returns {string} Unique request ID
   */
  generateRequestID() {
    // Simple UUID-like format: timestamp-counter-random
    const timestamp = Date.now().toString(36);
    const counter = (++this.requestIDCounter).toString(36);
    const random = Math.random().toString(36).substring(2, 7);
    return `${timestamp}-${counter}-${random}`;
  }

  /**
   * Format a log message with structured data
   * @private
   */
  _format(level, message, data = {}) {
    const timestamp = new Date().toISOString();
    const levelName = Object.keys(LogLevel).find(
      (key) => LogLevel[key] === level,
    );

    // Build structured log object
    const logData = {
      timestamp,
      level: levelName,
      message,
      ...data,
    };

    return logData;
  }

  /**
   * Output log to console with appropriate method
   * @private
   */
  _output(level, logData) {
    // Format: HH:MM:SS/L/C message | key=value
    // Where L = level initial (T/D/I/W/E), C = Client
    const time = new Date(logData.timestamp).toLocaleTimeString("en-US", {
      hour12: false,
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
    const levelInitial = logData.level.charAt(0); // T, D, I, W, or E
    const formatted = `${time}/${levelInitial}/C ${logData.message}`;

    const attrs = { ...logData };
    delete attrs.timestamp;
    delete attrs.level;
    delete attrs.message;

    // Convert to key=value format for readability
    const attrStr = Object.entries(attrs)
      .map(([key, value]) => `${key}=${JSON.stringify(value)}`)
      .join(" ");

    const fullMessage = attrStr ? `${formatted} | ${attrStr}` : formatted;

    // Choose console method based on level
    switch (level) {
      case LogLevel.TRACE:
      case LogLevel.DEBUG:
        console.debug(fullMessage, attrs);
        break;
      case LogLevel.INFO:
        console.log(fullMessage, attrs);
        break;
      case LogLevel.WARN:
        console.warn(fullMessage, attrs);
        break;
      case LogLevel.ERROR:
        console.error(fullMessage, attrs);
        break;
    }

    // Also buffer for backend if enabled
    this._bufferLog({
      timestamp: logData.timestamp,
      level: logData.level,
      message: logData.message,
      data: attrs,
    });
  }

  /**
   * Normalize arguments to structured data object
   * Handles both console.log-style args and structured data objects
   * @private
   */
  _normalizeArgs(args) {
    if (args.length === 0) {
      return {};
    }

    // If single argument is a plain object (not array, not null), use it directly
    if (
      args.length === 1 &&
      typeof args[0] === "object" &&
      args[0] !== null &&
      !Array.isArray(args[0])
    ) {
      return args[0];
    }

    // Otherwise, wrap all args in a data field
    // This handles primitives, arrays, and multiple arguments
    return { data: args.length === 1 ? args[0] : args };
  }

  /**
   * Log at TRACE level (very verbose, debug-time only)
   */
  trace(message, ...args) {
    if (this.level <= LogLevel.TRACE) {
      const data = this._normalizeArgs(args);
      const logData = this._format(LogLevel.TRACE, message, data);
      this._output(LogLevel.TRACE, logData);
    }
  }

  /**
   * Log at DEBUG level (internal component behavior)
   */
  debug(message, ...args) {
    if (this.level <= LogLevel.DEBUG) {
      const data = this._normalizeArgs(args);
      const logData = this._format(LogLevel.DEBUG, message, data);
      this._output(LogLevel.DEBUG, logData);
    }
  }

  /**
   * Log at INFO level (user-facing operations)
   */
  info(message, ...args) {
    if (this.level <= LogLevel.INFO) {
      const data = this._normalizeArgs(args);
      const logData = this._format(LogLevel.INFO, message, data);
      this._output(LogLevel.INFO, logData);
    }
  }

  /**
   * Log at WARN level (should be monitored)
   */
  warn(message, ...args) {
    if (this.level <= LogLevel.WARN) {
      const data = this._normalizeArgs(args);
      const logData = this._format(LogLevel.WARN, message, data);
      this._output(LogLevel.WARN, logData);
    }
  }

  /**
   * Log at ERROR level (logical bugs that shouldn't happen)
   */
  error(message, ...args) {
    if (this.level <= LogLevel.ERROR) {
      const data = this._normalizeArgs(args);
      const logData = this._format(LogLevel.ERROR, message, data);
      this._output(LogLevel.ERROR, logData);
    }
  }

  /**
   * Create a child logger with additional context
   * Useful for adding component name or request ID to all logs
   */
  child(contextData = {}) {
    const childLogger = Object.create(this);
    const originalFormat = this._format.bind(this);
    childLogger._format = (level, message, data = {}) => {
      return originalFormat(level, message, { ...contextData, ...data });
    };
    return childLogger;
  }
}

// Export singleton instance
const logger = new Logger();

// Also export LogLevel for configuration
window.LogLevel = LogLevel;
window.logger = logger;
