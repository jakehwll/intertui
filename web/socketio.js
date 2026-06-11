// Socket.IO bridge for the WASM client.
globalThis.intertuiNet = {
  _socket: null,
  _queue: [],
  _wake: null,

  setWake(fn) {
    this._wake = fn;
  },

  connect(url, callback) {
    const local =
      url.startsWith("http://localhost") ||
      url.startsWith("http://127.0.0.1");
    const socket = io(url, {
      transports: local ? ["polling"] : ["polling", "websocket"],
      reconnection: false,
    });
    this._socket = socket;

    socket.on("connect", () => callback(null));
    socket.on("connect_error", (err) => {
      callback(err && err.message ? err.message : String(err));
    });
    socket.on("data", (data) => {
      const raw = typeof data === "string" ? data : JSON.stringify(data);
      this._queue.push(raw);
      if (this._wake) {
        this._wake();
      }
    });
    socket.on("disconnect", () => {
      if (this._wake) {
        this._wake();
      }
    });
  },

  recv() {
    if (this._queue.length === 0) return "";
    return this._queue.shift();
  },

  send(raw) {
    if (!this._socket || !this._socket.connected) {
      return "socket.io not connected";
    }
    let out = raw;
    try {
      out = JSON.parse(raw);
    } catch (_) {
      // keep string payload
    }
    this._socket.emit("data", out);
    return null;
  },

  close() {
    if (this._socket) {
      this._socket.disconnect();
      this._socket = null;
    }
    this._queue = [];
    this._wake = null;
  },

  connected() {
    return !!(this._socket && this._socket.connected);
  },
};
