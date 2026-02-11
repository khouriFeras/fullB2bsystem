package domain

// OrderStatus represents the status of a supplier order (Shopify-aligned)
type OrderStatus string

const (
	// INCOMPLETE_CAUTION - New order, pending confirmation (was PENDING_CONFIRMATION)
	OrderStatusIncompleteCaution OrderStatus = "INCOMPLETE_CAUTION"
	// UNFULFILLED - Order confirmed, awaiting shipment (was CONFIRMED)
	OrderStatusUnfulfilled OrderStatus = "UNFULFILLED"
	// FULFILLED - Order shipped
	OrderStatusFulfilled OrderStatus = "FULFILLED"
	// COMPLETE - Order delivered
	OrderStatusComplete OrderStatus = "COMPLETE"
	// REJECTED - Supplier rejected the order
	OrderStatusRejected OrderStatus = "REJECTED"
	// CANCELED - Order canceled
	OrderStatusCanceled OrderStatus = "CANCELED"
	// REFUNDED - Order refunded
	OrderStatusRefunded OrderStatus = "REFUNDED"
	// ARCHIVED - Order archived
	OrderStatusArchived OrderStatus = "ARCHIVED"

	// Legacy aliases for backward compatibility (DB migration maps these)
	OrderStatusPendingConfirmation OrderStatus = "PENDING_CONFIRMATION"
	OrderStatusConfirmed           OrderStatus = "CONFIRMED"
	OrderStatusShipped             OrderStatus = "SHIPPED"
	OrderStatusDelivered           OrderStatus = "DELIVERED"
	OrderStatusCancelled           OrderStatus = "CANCELLED"
)

// IsValid checks if the order status is valid
func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusIncompleteCaution,
		OrderStatusUnfulfilled,
		OrderStatusFulfilled,
		OrderStatusComplete,
		OrderStatusRejected,
		OrderStatusCanceled,
		OrderStatusRefunded,
		OrderStatusArchived,
		// Legacy (accepted from DB until migrated)
		OrderStatusPendingConfirmation,
		OrderStatusConfirmed,
		OrderStatusShipped,
		OrderStatusDelivered,
		OrderStatusCancelled:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if a status transition is valid
func (s OrderStatus) CanTransitionTo(newStatus OrderStatus) bool {
	// Normalize legacy statuses for transition check
	from := s.normalize()
	to := newStatus.normalize()

	switch from {
	case OrderStatusIncompleteCaution:
		return to == OrderStatusUnfulfilled ||
			to == OrderStatusRejected ||
			to == OrderStatusCanceled
	case OrderStatusUnfulfilled:
		return to == OrderStatusFulfilled ||
			to == OrderStatusCanceled ||
			to == OrderStatusRefunded
	case OrderStatusFulfilled:
		return to == OrderStatusComplete ||
			to == OrderStatusRefunded
	case OrderStatusComplete:
		return to == OrderStatusRefunded ||
			to == OrderStatusArchived
	case OrderStatusRejected, OrderStatusCanceled, OrderStatusRefunded, OrderStatusArchived:
		return false // Terminal states
	default:
		return false
	}
}

// normalize maps legacy statuses to new ones for transition logic
func (s OrderStatus) normalize() OrderStatus {
	switch s {
	case OrderStatusPendingConfirmation:
		return OrderStatusIncompleteCaution
	case OrderStatusConfirmed:
		return OrderStatusUnfulfilled
	case OrderStatusShipped:
		return OrderStatusFulfilled
	case OrderStatusDelivered:
		return OrderStatusComplete
	case OrderStatusCancelled:
		return OrderStatusCanceled
	default:
		return s
	}
}
