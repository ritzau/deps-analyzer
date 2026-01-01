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
  }

  /**
   * Set the minimum log level
   * @param {number} level - LogLevel enum value
   */
  setLevel(level) {
    this.level = level;
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
    const levelName = Object.keys(LogLevel).find(key => LogLevel[key] === level);

    // Build structured log object
    const logData = {
      timestamp,
      level: levelName,
      message,
      ...data
    };

    return logData;
  }

  /**
   * Output log to console with appropriate method
   * @private
   */
  _output(level, logData) {
    const formatted = `[${logData.level}] ${logData.message}`;
    const attrs = { ...logData };
    delete attrs.timestamp;
    delete attrs.level;
    delete attrs.message;

    // Convert to key=value format for readability
    const attrStr = Object.entries(attrs)
      .map(([key, value]) => `${key}=${JSON.stringify(value)}`)
      .join(' ');

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
  }

  /**
   * Log at TRACE level (very verbose, debug-time only)
   */
  trace(message, data = {}) {
    if (this.level <= LogLevel.TRACE) {
      const logData = this._format(LogLevel.TRACE, message, data);
      this._output(LogLevel.TRACE, logData);
    }
  }

  /**
   * Log at DEBUG level (internal component behavior)
   */
  debug(message, data = {}) {
    if (this.level <= LogLevel.DEBUG) {
      const logData = this._format(LogLevel.DEBUG, message, data);
      this._output(LogLevel.DEBUG, logData);
    }
  }

  /**
   * Log at INFO level (user-facing operations)
   */
  info(message, data = {}) {
    if (this.level <= LogLevel.INFO) {
      const logData = this._format(LogLevel.INFO, message, data);
      this._output(LogLevel.INFO, logData);
    }
  }

  /**
   * Log at WARN level (should be monitored)
   */
  warn(message, data = {}) {
    if (this.level <= LogLevel.WARN) {
      const logData = this._format(LogLevel.WARN, message, data);
      this._output(LogLevel.WARN, logData);
    }
  }

  /**
   * Log at ERROR level (logical bugs that shouldn't happen)
   */
  error(message, data = {}) {
    if (this.level <= LogLevel.ERROR) {
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
