package logging

import (
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// WithComponent returns a child logger with the component field set.
func WithComponent(logger polylog.Logger, component string) polylog.Logger {
	return logger.With(FieldComponent, component)
}

// WithSupplier returns a child logger with the supplier field set.
func WithSupplier(logger polylog.Logger, supplierAddr string) polylog.Logger {
	return logger.With(FieldSupplier, supplierAddr)
}

// WithSession returns a child logger with the session_id field set.
func WithSession(logger polylog.Logger, sessionID string) polylog.Logger {
	return logger.With(FieldSessionID, sessionID)
}

// WithService returns a child logger with the service_id field set.
func WithService(logger polylog.Logger, serviceID string) polylog.Logger {
	return logger.With(FieldServiceID, serviceID)
}

// ForComponent returns a logger configured for a specific component.
// This is the preferred way to create component loggers.
func ForComponent(logger polylog.Logger, component string) polylog.Logger {
	return WithComponent(logger, component)
}

// ForSupplierComponent returns a logger configured for a supplier-specific component.
func ForSupplierComponent(logger polylog.Logger, component, supplierAddr string) polylog.Logger {
	return logger.With(
		FieldComponent, component,
		FieldSupplier, supplierAddr,
	)
}

// ForServiceComponent returns a logger configured for a service-specific component.
func ForServiceComponent(logger polylog.Logger, component, serviceID string) polylog.Logger {
	return logger.With(
		FieldComponent, component,
		FieldServiceID, serviceID,
	)
}

// ForSessionOperation returns a logger configured for session-specific operations.
func ForSessionOperation(logger polylog.Logger, sessionID string) polylog.Logger {
	return WithSession(logger, sessionID)
}
