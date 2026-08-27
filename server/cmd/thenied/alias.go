package main

import "log/slog"

// slogLogger keeps the command signatures short without importing log/slog into
// every one of them.
type slogLogger = slog.Logger
